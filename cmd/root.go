package cmd

import (
	"fmt"

	"github.com/TouchBistro/goutils/color"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/cmd/ios"
	"github.com/TouchBistro/tb/cmd/recipe"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/fortune"
	"github.com/TouchBistro/tb/git"
	"github.com/blang/semver"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var version string
var rootOpts struct {
	noUpdateRecipes bool
}

var rootCmd = &cobra.Command{
	Use:     "tb",
	Version: version,
	Short:   "tb is a CLI for running TouchBistro services on a development machine",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initOpts := config.InitOptions{
			UpdateRecipes: !rootOpts.noUpdateRecipes,
			LegacyInit:    true,
		}
		baseCmdName := cmd.Parent().Name()

		if baseCmdName == "ios" {
			initOpts.LoadApps = true
		} else if baseCmdName == "tb" {
			initOpts.LoadServices = true
		}

		err := config.Init(initOpts)
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
	rootCmd.PersistentFlags().BoolVar(&rootOpts.noUpdateRecipes, "no-update-recipes", false, "Don't update recipes when tb is run")

	// Add subcommands
	rootCmd.AddCommand(ios.IOS())
	rootCmd.AddCommand(recipe.Recipe())

	cobra.OnInitialize(func() {
		f := fortune.Random().String()
		fmt.Println(color.Magenta(f))
		checkVersion()
	})
}

func checkVersion() {
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
