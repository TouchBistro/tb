package commands

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"

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

			if err := checkVersion(cmd.Context(), version, c.Tracker); err != nil {
				c.Tracker.Debug(fmt.Sprintf("Ignoring error encountered while checking version of tb: %v", err))
			}

			if err := checkDepVersion(cmd.Context(), c.Tracker); err != nil {
				c.Tracker.Debug(fmt.Sprintf("Ignoring error encountered while checking dependency versions: %v", err))
			}

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

func checkVersion(ctx context.Context, version string, logger progress.Logger) error {
	currentVersion, err := semver.Parse(version)
	if err != nil {
		return fmt.Errorf("unable to parse current tb version: %w", err)
	}

	// Check if there is a newer version available and let the user know
	// If it fails just ignore and continue normal operation
	// Log to debug for troubleshooting
	githubClient := github.New(&http.Client{})
	latestRelease, err := githubClient.LatestReleaseTag(ctx, "TouchBistro", "tb")
	if err != nil {
		return fmt.Errorf("failed to get latest version of tb from GitHub: %w", err)
	}
	latestVersion, err := semver.Parse(latestRelease)
	if err != nil {
		return fmt.Errorf("unable to parse latest tb version: %w", err)
	}
	if !currentVersion.LT(latestVersion) {
		return nil
	}

	logger.Info(color.Yellow("🚨🚨🚨 Your version of tb is out of date 🚨🚨🚨"))
	logger.Infof("%s: %s", color.Yellow("Current version"), color.Cyan(version))
	logger.Infof("%s: %s", color.Yellow("Latest version"), color.Cyan(latestRelease))
	logger.Infof("%s: %s", color.Yellow("Please consider upgrading by running"), color.Cyan("brew update && brew upgrade tb"))

	// Tell people to stay safe if major version
	if latestVersion.Major > currentVersion.Major {
		logger.Info(color.Red("🚨🚨🚨 WARNING: This is a major version upgrade 🚨🚨🚨"))
		logger.Info(color.Red("Please upgrade with caution."))
	}

	return nil
}

func checkDepVersion(ctx context.Context, logger progress.Logger) error {
	dependencies := map[string]struct {
		Command []string
		Repo    string
	}{
		"docker-engine": {Command: []string{"docker", "version", "--format", "{{.Client.Version}}"}, Repo: "moby/moby"},
		// docker-compose is now integrated into the docker CLI as 'docker compose'
		"lazydocker": {Command: []string{"lazydocker", "--version"}, Repo: "jesseduffield/lazydocker"},
	}
	re := regexp.MustCompile(`\d+\.\d+\.\d+`)

	for name, dep := range dependencies {
		cmd := exec.Command(dep.Command[0], dep.Command[1:]...)
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to check %s version: %v", name, err)
		}

		version := re.FindString(string(output))
		currentVersion, err := semver.Parse(version)
		if err != nil {
			return fmt.Errorf("unable to parse %s version: %v", name, err)
		}

		// Special check for lazydocker - Docker SDK had breaking changes in November 2025
		if name == "lazydocker" {
			minRequiredVersion := semver.MustParse("0.24.2")
			if currentVersion.LT(minRequiredVersion) {
				logger.Info("")
				logger.Info(color.Red("🚨 Your lazydocker version (%s) is too old for the Docker SDK changes in November 2025."), version)
				logger.Info(color.Red("   lazydocker v0.24.2 or newer is required."))
				logger.Info(color.Yellow("   Please upgrade: brew upgrade lazydocker"))
				logger.Info("")
			}
		}

		// Check if there is a newer version available and let the user know
		// If it fails just ignore and continue normal operation
		// Log to debug for troubleshooting
		githubClient := github.New(&http.Client{})
		paths := strings.Split(dep.Repo, "/")
		latestRelease, err := githubClient.LatestReleaseTag(ctx, paths[0], paths[1])
		if err != nil {
			return fmt.Errorf("failed to get latest %s version from GitHub: %v", name, err)
		}
		latestVersion, err := semver.ParseTolerant(latestRelease)
		if err != nil {
			return fmt.Errorf("unable to parse latest %s version: %v", name, err)
		}
		if !currentVersion.LT(latestVersion) {
			continue
		}

		logger.Infof(color.Yellow(fmt.Sprintf("🚨 Your version of %s is out of date 🚨", name)))
		logger.Infof("%s: %s. %s: %s", color.Yellow("Current"), color.Cyan(version), color.Yellow("Latest"), color.Cyan(latestRelease))
	}

	return nil
}
