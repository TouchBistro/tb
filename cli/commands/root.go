package commands

import (
	"context"
	"net/http"

	"github.com/TouchBistro/goutils/color"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/cli"
	appCommands "github.com/TouchBistro/tb/cli/commands/app"
	registryCommands "github.com/TouchBistro/tb/cli/commands/registry"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/integrations/github"
	"github.com/TouchBistro/tb/util"
	"github.com/blang/semver/v4"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	noRegistryPull bool
	verbose        bool
}

func NewRootCommand(c *cli.Container, version string) *cobra.Command {
	var opts rootOptions
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
			// Get the user config, pass empty string to have it find the config file
			cfg, err := config.Read("")
			if err != nil {
				return &cli.ExitError{
					Message: "Failed to load tbrc",
					Err:     err,
				}
			}
			c.Verbose = opts.verbose || cfg.DebugEnabled()

			// Initialize logging
			c.Logger, err = cli.NewLogger(c.Verbose)
			if err != nil {
				return err
			}
			c.Tracker = &progress.SpinnerTracker{
				OutputLogger:    c.Logger,
				PersistMessages: c.Verbose,
			}

			// Any special messages based on user config
			if cfg.Debug != nil {
				// This prints a warning sign
				c.Tracker.Warn("\u26A0\uFE0F  Using the 'debug' field in tbrc.yml is deprecated. Use the '--verbose' or '-v' flag instead.")
			}
			if cfg.ExperimentalMode {
				c.Tracker.Info(color.Yellow("🚧 Experimental mode enabled 🚧"))
				c.Tracker.Info(color.Yellow("If you find any bugs please report them in an issue: https://github.com/TouchBistro/tb/issues"))
			}
			checkVersion(cmd.Context(), version, c.Tracker)

			// Determine how to proceed based on the type of command
			initOpts := config.InitOptions{UpdateRegistries: !opts.noRegistryPull}
			switch cmd.Parent().Name() {
			case "registry":
				// No further action required for registry commands
				return nil
			case "ios":
				if !util.IsMacOS {
					return &cli.ExitError{Message: "tb app ios is only supported on macOS"}
				}
				fallthrough
			case "app", "desktop":
				initOpts.LoadApps = true
			default:
				initOpts.LoadServices = true
			}

			// Create the context that commands can use.
			// Generally it is recommended not to store contexts in structs, however this case is special
			// since only one command runs on the each invocation of tb and the container can be seen
			// as special parameters to the command. Also cobra does with cmd.Context().
			c.Ctx = progress.ContextWithTracker(cmd.Context(), c.Tracker)
			c.Engine, err = config.Init(c.Ctx, cfg, initOpts)
			if err != nil {
				return &cli.ExitError{
					Message: "Failed to load registries",
					Err:     err,
				}
			}
			return nil
		},
	}

	persistentFlags := rootCmd.PersistentFlags()
	persistentFlags.BoolVar(&opts.noRegistryPull, "no-registry-pull", false, "Don't pull latest version of registries when tb is run")
	persistentFlags.BoolVarP(&opts.verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.AddCommand(
		appCommands.NewAppCommand(c),
		registryCommands.NewRegistryCommand(c),
		newCloneCommand(c),
		newCompletionsCommand(),
		newDBCommand(c),
		newDownCommand(c),
		newExecCommand(c),
		newImagesCommand(c),
		newListCommand(c),
		newLogsCommand(c),
		newNukeCommand(c),
		newUpCommand(c),
	)
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
