package commands

import (
	"context"
	"net/http"

	"github.com/TouchBistro/goutils/color"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/cli"
	appCommands "github.com/TouchBistro/tb/cli/commands/app"
	registryCommands "github.com/TouchBistro/tb/cli/commands/registry"
	"github.com/TouchBistro/tb/cmd"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/integrations/github"
	"github.com/TouchBistro/tb/util"
	"github.com/blang/semver"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewRootCommand(c *cli.Container, version string) *cobra.Command {
	var rootOpts struct {
		noRegistryPull bool
		verbose        bool
	}
	rootCmd := &cobra.Command{
		Use:     "tb",
		Version: version,
		Short:   "tb is a CLI for running services on a development machine",
		CompletionOptions: cobra.CompletionOptions{
			// Cobra generates an `completion` command by default.
			// Disable this since we handle completions ourselves.
			DisableDefaultCmd: true,
		},
		// cobra prints errors returned from RunE by default. Disable that since we handle errors ourselves.
		SilenceErrors: true,
		// cobra prints command usage by default if RunE returns an error.
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Initialize logging
			if rootOpts.verbose {
				logrus.SetLevel(logrus.DebugLevel)
				fatal.PrintDetailedError(true)
			}
			logrus.SetFormatter(&logrus.TextFormatter{
				DisableTimestamp: true,
				// Need to force colours since the decision of whether or not to use colour
				// is made lazily the first time a log is written, and Out may be changed
				// to a spinner before then.
				ForceColors: true,
			})
			if err := config.LoadTBRC(); err != nil {
				return &cli.ExitError{
					Message: "Failed to load tbrc",
					Err:     err,
				}
			}
			c.Verbose = rootOpts.verbose || config.IsDebugEnabled()
			c.Tracker = &progress.SpinnerTracker{
				OutputLogger:    cli.OutputLogger{Logger: logrus.StandardLogger()},
				PersistMessages: c.Verbose,
			}
			checkVersion(cmd.Context(), version, c.Tracker)

			// Determine how to proceed based on the type of command
			initOpts := config.InitOptions{UpdateRegistries: !rootOpts.noRegistryPull}
			switch cmd.Parent().Name() {
			case "registry":
				// No further action required for registry commands
				return nil
			case "ios":
				if !util.IsMacOS() {
					return &cli.ExitError{Message: "tb app ios is only supported on macOS"}
				}
				fallthrough
			case "app", "desktop":
				initOpts.LoadApps = true
			default:
				initOpts.LoadServices = true
			}

			ctx := progress.ContextWithTracker(cmd.Context(), c.Tracker)
			if err := config.Init(ctx, initOpts); err != nil {
				return &cli.ExitError{
					Message: "Failed to load registries",
					Err:     err,
				}
			}
			c.Engine = config.Engine()
			return nil
		},
	}
	rootCmd.PersistentFlags().BoolVar(&rootOpts.noRegistryPull, "no-registry-pull", false, "Don't pull latest version of registries when tb is run")
	rootCmd.PersistentFlags().BoolVarP(&rootOpts.verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.AddCommand(
		appCommands.NewAppCommand(c),
		registryCommands.NewRegistryCommand(c),
		newDownCommand(c),
	)
	// Add legacy style commands
	rootCmd.AddCommand(cmd.Commands()...)
	return rootCmd
}

func checkVersion(ctx context.Context, version string, logger progress.Logger) {
	currentVersion, err := semver.Parse(version)
	if err != nil {
		logger.Debug("Unable to check current version of tb")
		return
	}

	// Check if there is a newer version available and let the user know
	// If it fails just ignore and continue normal operation
	// Log to debug for troubleshooting
	githubClient := github.New(&http.Client{})
	latestRelease, err := githubClient.LatestReleaseTag(ctx, "TouchBistro", "tb")
	if err != nil {
		logger.Debug("Failed to get latest version of tb from GitHub. Skipping.")
		logger.Debug(err)
		return
	}
	latestVersion, err := semver.Parse(latestRelease)
	if err != nil {
		logger.Debug("Unable to check latest version of tb")
		return
	}
	if !currentVersion.LT(latestVersion) {
		return
	}

	logger.Info(color.Yellow("🚨🚨🚨 Your version of tb is out of date 🚨🚨🚨"))
	logger.Info(color.Yellow("Current version: "), color.Cyan(version))
	logger.Info(color.Yellow("Latest version: "), color.Cyan(latestRelease))
	logger.Info(color.Yellow("Please consider upgrading by running: "), color.Cyan("brew update && brew upgrade tb"))

	// Tell people to stay safe if major version
	if latestVersion.Major > currentVersion.Major {
		logger.Info(color.Red("🚨🚨🚨 WARNING: This is a major version upgrade 🚨🚨🚨"))
		logger.Info(color.Red("Please upgrade with caution."))
	}
}
