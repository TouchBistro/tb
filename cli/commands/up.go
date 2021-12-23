package commands

import (
	"context"

	"github.com/TouchBistro/goutils/command"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/deps"
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
			if err := deps.Resolve(ctx, deps.Brew, deps.Lazydocker); err != nil {
				return err
			}
			loginStrategies, err := config.LoginStategies()
			if err != nil {
				return err
			}
			if len(loginStrategies) > 0 {
				err := progress.RunParallel(ctx, progress.RunParallelOptions{
					Message: "Logging into services",
					Count:   len(loginStrategies),
					// Bail if one fails since there's no point on waiting on the others
					// since we can't proceed anyway.
					CancelOnError: true,
				}, func(ctx context.Context, i int) error {
					ls := loginStrategies[i]
					tracker := progress.TrackerFromContext(ctx)
					tracker.Debugf("Logging into %s", ls.Name())
					if err := ls.Login(ctx); err != nil {
						return err
					}
					tracker.Debugf("Logged into %s", ls.Name())
					return nil
				})
				if err != nil {
					return err
				}
				c.Tracker.Debug("Finished logging into services")
			}

			err = c.Engine.Up(ctx, engine.UpOptions{
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

			if !opts.skipLazydocker {
				c.Tracker.Debug("Running lazydocker")
				w := progress.LogWriter(c.Tracker, c.Tracker.WithFields(progress.Fields{"id": "lazydocker"}).Debug)
				defer w.Close()
				err := command.New(command.WithStdout(w), command.WithStderr(w)).Exec("lazydocker")
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
