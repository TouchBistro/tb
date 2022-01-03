package commands

import (
	"fmt"

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
		Use: "up [services...]",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && opts.playlistName == "" && len(opts.serviceNames) == 0 {
				return fmt.Errorf("service names or playlist name is required")
			}
			if len(args) > 0 && opts.playlistName != "" {
				return fmt.Errorf("cannot specify service names as args when --playlist or -p is used")
			}
			// This is deprecated and will be removed but we need to check for it for now
			if len(args) > 0 && len(opts.serviceNames) > 0 {
				return fmt.Errorf("cannot specify service names as args when --services or -s is used")
			}
			return nil
		},
		Short: "Start services or playlists",
		Long: `Starts one or more services. The following actions will be performed before starting services:

- Stop and remove any services that are already running.
- Pull base images and service images.
- Build any services with mode build.
- Run pre-run steps for services.

Services can be specified in one of two ways. First, the names of the services can be specified directly as args.
Second, the --playlist,-p flag can be used to provide a playlist name in order to start all the services in the playlist.
If a playlist is provided no args can be provided, that is, mixing a playlist and service names is not allowed.

Examples:

Run the services defined in the 'core' playlist in a registry:

	tb up --playlist core

Run the postgres and localstack services directly:

	tb up postgres localstack`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Hack to support either args or --services flag for backwards compatibility.
			// The flag will eventually be removed so we won't have to do this
			// and will be able to just pass args to Engine.Up.
			serviceNames := args
			if len(serviceNames) == 0 {
				serviceNames = opts.serviceNames
			}
			ctx := progress.ContextWithTracker(cmd.Context(), c.Tracker)
			err := c.Engine.Up(ctx, engine.UpOptions{
				ServiceNames:   serviceNames,
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
	flags.BoolVar(&opts.skipServicePreRun, "no-service-prerun", false, "Don't run preRun command for services")
	flags.BoolVar(&opts.skipGitPull, "no-git-pull", false, "Don't update git repositories")
	flags.BoolVar(&opts.skipDockerPull, "no-remote-pull", false, "Don't get new remote images")
	flags.BoolVar(&opts.skipLazydocker, "no-lazydocker", false, "Don't start lazydocker")
	flags.StringVarP(&opts.playlistName, "playlist", "p", "", "The name of a playlist")
	flags.StringSliceVarP(&opts.serviceNames, "services", "s", []string{}, "Comma separated list of services to start. eg --services postgres,localstack.")
	err := flags.MarkDeprecated("services", "it is a no-op and will be removed")
	if err != nil {
		// MarkDeprecated only errors if the flag name is wrong or the message isn't set
		// which is a programming error, so we wanna blow up
		panic(err)
	}
	return upCmd
}
