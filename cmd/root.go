package cmd

import (
	"fmt"

	"github.com/TouchBistro/tb/cmd/ios"
	"github.com/TouchBistro/goutils/color"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/fortune"
	"github.com/TouchBistro/tb/git"
	"github.com/blang/semver"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var version string

var rootCmd = &cobra.Command{
	Use:     "tb",
	Version: version,
	Short:   "tb is a CLI for running TouchBistro services on a development machine",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fatal.ExitErr(err, "Failed executing command.")
	}
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(ios.IOS())

	cobra.OnInitialize(func() {
		f := fortune.Random().String()
		fmt.Println(color.Magenta(f))
		initConfig()
	})
}

func initConfig() {
	err := config.InitRC()
	if err != nil {
		fatal.ExitErr(err, "Failed to initialise .tbrc file.")
	}

	var logLevel log.Level
	if config.TBRC().DebugEnabled {
		logLevel = log.DebugLevel
	} else {
		logLevel = log.InfoLevel
	}

	log.SetLevel(logLevel)
	log.SetFormatter(&log.TextFormatter{
		// TODO: Remove the log level - its quite ugly
		DisableTimestamp: true,
	})

	if logLevel != log.DebugLevel {
		fatal.ShowStackTraces = false
	}

	err = config.Init()
	if err != nil {
		fatal.ExitErr(err, "Failed to initialise config files.")
	}

	checkVersion()
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
