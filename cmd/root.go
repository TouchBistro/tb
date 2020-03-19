package cmd

import (
	"fmt"

	"github.com/TouchBistro/goutils/color"
	"github.com/TouchBistro/goutils/fatal"
	appCmd "github.com/TouchBistro/tb/cmd/app"
	"github.com/TouchBistro/tb/cmd/app/desktop"
	"github.com/TouchBistro/tb/cmd/app/ios"
	legacyIOSCmd "github.com/TouchBistro/tb/cmd/ios"
	registryCmd "github.com/TouchBistro/tb/cmd/registry"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/fortune"
	"github.com/TouchBistro/tb/git"
	"github.com/blang/semver"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var version string
var rootOpts struct {
	noRegistryPull bool
}

var rootCmd = &cobra.Command{
	Use:     "tb",
	Version: version,
	Short:   "tb is a CLI for running TouchBistro services on a development machine",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := config.Init(config.InitOptions{
			UpdateRegistries: !rootOpts.noRegistryPull,
			LoadServices:     true,
			LoadApps:         false,
		})
		if err != nil {
			fatal.ExitErr(err, "Failed to initialise config files.")
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fatal.ExitErr(err, "Failed executing command.")
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&rootOpts.noRegistryPull, "no-registry-pull", false, "Don't pull latest version of registries when tb is run")

	// Add subcommands
	appCmd.AppCmd().AddCommand(desktop.DesktopCmd(), ios.IOSCmd())
	rootCmd.AddCommand(appCmd.AppCmd(), legacyIOSCmd.IOS(), registryCmd.RegistryCmd())

	cobra.OnInitialize(func() {
		f := fortune.Random().String()
		fmt.Println(color.Magenta(f))

		err := config.LoadTBRC()
		if err != nil {
			fatal.ExitErr(err, "Failed to load tbrc.")
		}

		if config.IsExperimentalEnabled() {
			log.Infoln(color.Yellow("🚧 Experimental mode enabled 🚧"))
			log.Infoln(color.Yellow("If you find any bugs please report them in an issue: https://github.com/TouchBistro/tb/issues"))
		}

		checkVersion()
	})
}

func checkVersion() {
	// If version isn't set it means it was built manually so don't bother checking version
	if version == "" {
		log.Info(color.Blue("tb was built from source. Skipping latest version check."))
		return
	}

	// Check if there is a newer version available and let the user know
	// If it fails just ignore and continue normal operation
	// Log to debug for troubleshooting
	latestRelease, err := git.GetLatestRelease()
	if err != nil {
		log.Debugln("Failed to get latest version of tb from GitHub. Skipping.")
		log.Debugln(err)
		return
	}

	currentVersion, err := semver.Make(version)
	if err != nil {
		log.Debugln("Unable to check current version of tb")
		return
	}

	latestVersion, err := semver.Make(latestRelease)
	if err != nil {
		log.Debugln("Unable to check latest version of tb")
		return
	}

	isLessThan := currentVersion.LT(latestVersion)
	if !isLessThan {
		return
	}

	log.Info(color.Yellow("🚨🚨🚨 Your version of tb is out of date 🚨🚨🚨"))
	log.Info(color.Yellow("Current version: "), color.Cyan(version))
	log.Info(color.Yellow("Latest version: "), color.Cyan(latestRelease))
	log.Info(color.Yellow("Please consider upgrading by running: "), color.Cyan("brew update && brew upgrade tb"))

	// Tell people to stay safe if major version
	if latestVersion.Major > currentVersion.Major {
		log.Info(color.Red("🚨🚨🚨 WARNING: This is a major version upgrade 🚨🚨🚨"))
		log.Info(color.Red("Please upgrade with caution."))
	}
}

func Root() *cobra.Command {
	return rootCmd
}
