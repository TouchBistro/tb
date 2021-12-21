package commands

import (
	"context"

	"github.com/TouchBistro/goutils/command"
	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/deps"
	"github.com/TouchBistro/tb/engine"
	"github.com/TouchBistro/tb/resource"
	"github.com/TouchBistro/tb/resource/service"
	"github.com/spf13/cobra"
)

func newUpCommand(c *cli.Container) *cobra.Command {
	var upOpts struct {
		skipServicePreRun bool
		skipGitPull       bool
		skipDockerPull    bool
		skipLazydocker    bool
		playlistName      string
		serviceNames      []string
	}
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
			var services []service.Service
			var err error
			switch {
			case len(upOpts.serviceNames) > 0 && upOpts.playlistName != "":
				return &cli.ExitError{
					Message: "Only one of --playlist or --services can be specified.\nTry tb up --help for some examples.",
				}
			case len(upOpts.serviceNames) > 0:
				services, err = c.Engine.ResolveServices(upOpts.serviceNames)
			case upOpts.playlistName != "":
				services, err = c.Engine.ResolvePlaylist(upOpts.playlistName)
			default:
				return &cli.ExitError{
					Message: "Either --playlist or --services must be specified.\ntry tb up --help for some examples.",
				}
			}
			if errors.Is(err, resource.ErrNotFound) {
				return &cli.ExitError{
					Message: "Try running `tb list` to see available services and playlists",
					Err:     err,
				}
			} else if err != nil {
				return err
			}

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

			err = c.Engine.Up(ctx, services, engine.UpOptions{
				SkipPreRun:     upOpts.skipServicePreRun,
				SkipDockerPull: upOpts.skipDockerPull,
				SkipGitPull:    upOpts.skipGitPull,
			})
			if err != nil {
				return &cli.ExitError{
					Message: "Failed to stop services",
					Err:     err,
				}
			}
			c.Tracker.Info("✔ Started services")

			if !upOpts.skipLazydocker {
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
	upCmd.Flags().BoolVar(&upOpts.skipServicePreRun, "no-service-prerun", false, "dont run preRun command for services")
	upCmd.Flags().BoolVar(&upOpts.skipGitPull, "no-git-pull", false, "dont update git repositories")
	upCmd.Flags().BoolVar(&upOpts.skipDockerPull, "no-remote-pull", false, "dont get new remote images")
	upCmd.Flags().BoolVar(&upOpts.skipLazydocker, "no-lazydocker", false, "dont start lazydocker")
	upCmd.Flags().StringVarP(&upOpts.playlistName, "playlist", "p", "", "the name of a service playlist")
	upCmd.Flags().StringSliceVarP(&upOpts.serviceNames, "services", "s", []string{}, "comma separated list of services to start. eg --services postgres,localstack.")
	return upCmd
}
