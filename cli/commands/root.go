package commands

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/TouchBistro/goutils/color"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/logutil"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/goutils/spinner"
	"github.com/TouchBistro/tb/cli"
	appCommands "github.com/TouchBistro/tb/cli/commands/app"
	registryCommands "github.com/TouchBistro/tb/cli/commands/registry"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/integrations/github"
	"github.com/TouchBistro/tb/internal/fortune"
	"github.com/TouchBistro/tb/internal/util"
	"github.com/blang/semver/v4"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type rootOptions struct {
	noRegistryPull bool
	verbose        bool
	offlineMode    bool
}

func NewRootCommand(c *cli.Container, version string) *cobra.Command {
	var opts rootOptions
	rootCmd := &cobra.Command{
		Use:     "tb",
		Version: version,
		Short:   "tb is a CLI for running services on a development machine",
		// cobra prints errors returned from RunE by default. Disable that since we handle errors ourselves.
		SilenceErrors: true,
		// cobra prints command usage by default if RunE returns an error.
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Check if the command being run is one of the completion commands provided by cobra.
			// If so, skip all initialization since it unnecessary overhead.
			if cmd.Parent().Name() == "completion" {
				return nil
			}

			// Print out fortune first
			// Figure out the size of the terminal
			termWidth, _, err := term.GetSize(int(os.Stderr.Fd()))
			if err != nil {
				// Likely means it isn't a terminal, just pass 0 and the fortune
				// will do the right thing
				termWidth = 0
			}
			fmt.Fprintln(os.Stderr, color.Magenta(fortune.Random().Pretty(termWidth)))

			// Get the user config, pass empty string to have it find the config file
			cfg, err := config.Read("")
			if err != nil {
				return &fatal.Error{
					Msg: "Failed to load tbrc",
					Err: err,
				}
			}
			c.Verbose = opts.verbose || cfg.DebugEnabled()
			c.OfflineMode = opts.offlineMode

			// Initialize logging
			// Create a temp file to log to.
			c.Logfile, err = os.CreateTemp("", "tb_log_*.txt")
			if err != nil {
				return &fatal.Error{Msg: "Failed to create log file", Err: err}
			}
			level := slog.LevelInfo
			if c.Verbose {
				level = slog.LevelDebug
			}
			c.Tracker = spinner.NewTracker(spinner.TrackerOptions{
				PersistMessages: c.Verbose,
				NewHandler: func(w io.Writer) slog.Handler {
					return logutil.NewMultiHandler([]slog.Handler{
						logutil.NewPrettyHandler(w, &logutil.PrettyHandlerOptions{
							Level:       level,
							ReplaceAttr: logutil.RemoveKeys(slog.TimeKey),
						}),
						slog.NewTextHandler(c.Logfile, &slog.HandlerOptions{Level: slog.LevelDebug}),
					}, nil)
				},
			})

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
			initOpts := config.InitOptions{UpdateRegistries: !opts.noRegistryPull && !opts.offlineMode}
			switch cmd.Parent().Name() {
			case "registry":
				// No further action required for registry commands
				return nil
			case "ios":
				if !util.IsMacOS {
					return &fatal.Error{Msg: "tb app ios is only supported on macOS"}
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
				return &fatal.Error{
					Msg: "Failed to load registries",
					Err: err,
				}
			}
			return nil
		},
	}

	persistentFlags := rootCmd.PersistentFlags()
	persistentFlags.BoolVar(&opts.noRegistryPull, "no-registry-pull", false, "Don't pull latest version of registries when tb is run")
	persistentFlags.BoolVarP(&opts.offlineMode, "offline", "o", false, "Skip operations requiring internet connectivity")
	persistentFlags.BoolVarP(&opts.verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.AddCommand(
		appCommands.NewAppCommand(c),
		registryCommands.NewRegistryCommand(c),
		newCloneCommand(c),
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
		logger.WithAttrs("err", err).Debug("Failed to get latest version of tb from GitHub. Skipping.")
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
