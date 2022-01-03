package commands

import (
	"github.com/TouchBistro/goutils/command"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/engine"
	"github.com/spf13/cobra"
)

type upOptions struct {
	skipServicePreRun bool
	skipGitPull       bool
	skipDockerPull    bool
	skipLazydocker    bool
	playlistName      string
	serviceNames      []string
}

func newUpCommand(c *cli.Container) *cobra.Command {
	var opts upOptions
	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Starts services from a playlist name or as a comma separated list of services",
		Long: `Starts services from a playlist name or as a comma separated list of services.

	Examples:
	- run the services defined under the "core" key in playlists.yml
		tb up --playlist core

	- run only postgres and localstack
		tb up --services postgres,localstack`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := progress.ContextWithTracker(cmd.Context(), c.Tracker)
			err := c.Engine.Up(ctx, engine.UpOptions{
				ServiceNames:   opts.serviceNames,
				PlaylistName:   opts.playlistName,
				SkipPreRun:     opts.skipServicePreRun,
				SkipDockerPull: opts.skipDockerPull,
				SkipGitPull:    opts.skipGitPull,
			})
			if err != nil {
				return &cli.ExitError{
					Message: "Failed to start services",
					Err:     err,
				}
			}
			c.Tracker.Info("✔ Started services")

			// lazydocker opt in, if it exists it will be launched, otherwise this step will be skipped
			const lazydocker = "lazydocker"
			if !opts.skipLazydocker && command.IsAvailable(lazydocker) {
				c.Tracker.Debug("Running lazydocker")
				w := progress.LogWriter(c.Tracker, c.Tracker.WithFields(progress.Fields{"id": "lazydocker"}).Debug)
				defer w.Close()
				err := command.New(command.WithStdout(w), command.WithStderr(w)).Exec(lazydocker)
				if err != nil {
					return &cli.ExitError{
						Message: "Failed running lazydocker",
						Err:     err,
					}
				}
			}
			c.Tracker.Info("🔈 the containers are running in the background. If you want to terminate them, run tb down")
			return nil
		},
	}

	flags := upCmd.Flags()
	flags.BoolVar(&opts.skipServicePreRun, "no-service-prerun", false, "dont run preRun command for services")
	flags.BoolVar(&opts.skipGitPull, "no-git-pull", false, "dont update git repositories")
	flags.BoolVar(&opts.skipDockerPull, "no-remote-pull", false, "dont get new remote images")
	flags.BoolVar(&opts.skipLazydocker, "no-lazydocker", false, "dont start lazydocker")
	flags.StringVarP(&opts.playlistName, "playlist", "p", "", "the name of a service playlist")
	flags.StringSliceVarP(&opts.serviceNames, "services", "s", []string{}, "comma separated list of services to start. eg --services postgres,localstack.")
	return upCmd
}
