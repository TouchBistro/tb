package cmd

import (
	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/docker"
	"github.com/TouchBistro/tb/resource"
	"github.com/TouchBistro/tb/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type downOptions struct {
	shouldSkipGitPull bool
}

var downOpts downOptions

var downCmd = &cobra.Command{
	Use:   "down [services...]",
	Short: "Stop and remove containers",
	Run: func(cmd *cobra.Command, args []string) {
		// DISCUSS(@cszatmary): Not sure why we did this but we call compose stop & compose rm
		// instead of calling compose down. down also removes any networks created which we don't
		// This is probably why we've seen spooky network stuff before.

		// DISCUSS(@cszatmary): Does it makes sense to allow stopping services that don't exist in
		// registries anymore? Ex: a service is removed, but you still have it running locally?
		// It seems to make more sense to not allow services that have been removed. However, this
		// means you could get into weird states where a service was removed from the registry and
		// now you have no way to stop tb without manually docker commands.

		if config.IsExperimentalEnabled() {
			eng := config.Engine()
			services, err := eng.ResolveServices(args)
			if errors.Is(err, resource.ErrNotFound) {
				fatal.ExitErr(err, "Try running `tb list` to see available services")
			} else if err != nil {
				fatal.ExitErr(err, "Failed to resolve services to stop")
			}

			tracker := &progress.SpinnerTracker{
				OutputLogger:    util.OutputLogger{Logger: log.StandardLogger()},
				PersistMessages: config.IsDebugEnabled(),
			}
			ctx := progress.ContextWithTracker(cmd.Context(), tracker)
			if err := eng.Down(ctx, services); err != nil {
				fatal.ExitErr(err, "Failed to stop services")
			}
			log.Info("✔ Cleaned up previous docker state")
			return
		}

		// DISCUSS(@cszatmary): Scope if there's a way we can avoid doing this, it seems unnecessary
		// and kinda blows.
		// If we must have it, scope only cloning repos that are missing that are required.
		// There's no need to clone/pull every single repo if you are only stopping 2 services for ex.
		err := config.CloneOrPullRepos(!downOpts.shouldSkipGitPull)
		if err != nil {
			fatal.ExitErr(err, "failed cloning git repos.")
		}

		log.Debug("stopping compose services...")

		names := make([]string, len(args))
		for _, serviceName := range args {
			s, err := config.LoadedServices().Get(serviceName)
			if err != nil {
				fatal.ExitErrf(err, "%s is not a valid service\n. Try running `tb list` to see available services\n", serviceName)
			}

			names = append(names, util.DockerName(s.FullName()))
		}

		err = docker.ComposeStop(names)
		if err != nil {
			fatal.ExitErr(err, "failed stopping compose services")
		}
		log.Debug("...done")
		if err != nil {
			fatal.ExitErr(err, "could not stop containers and services")
		}

		log.Println("removing stopped containers...")
		err = docker.ComposeRm(names)
		if err != nil {
			fatal.ExitErr(err, "could not remove stopped containers")
		}
		log.Println("done")
	},
}

func init() {
	downCmd.Flags().BoolVar(&downOpts.shouldSkipGitPull, "no-git-pull", false, "dont update git repositories")

	rootCmd.AddCommand(downCmd)
}
