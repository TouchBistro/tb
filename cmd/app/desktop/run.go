package desktop

import (
	"errors"
	"os"

	"github.com/TouchBistro/goutils/command"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/file"
	appCmd "github.com/TouchBistro/tb/cmd/app"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type runOptions struct {
	branch string
}

var runOpts runOptions

var runCmd = &cobra.Command{
	Use: "run",
	Args: func(cmd *cobra.Command, args []string) error {
		// Verify that the app name was provided as a single arg
		if len(args) < 1 {
			return errors.New("app name is required as an argument")
		} else if len(args) > 1 {
			return errors.New("only one argument is accepted")
		}

		return nil
	},
	Short: "Runs a desktop application",
	Long: `Runs a desktop application.

Examples:
- run the current master build of TouchBistroServer
  tb app desktop run TouchBistroServer

- run the build for a specific branch
  tb app desktop run TouchBistroServer --branch task/bug-631/fix-thing`,
	Run: func(cmd *cobra.Command, args []string) {
		appName := args[0]
		a, err := config.LoadedDesktopApps().Get(appName)
		if err != nil {
			fatal.ExitErrf(err, "%s is not a valid desktop app\n", appName)
		}

		// Override branch if one was provided
		if runOpts.branch != "" {
			a.Branch = runOpts.branch
		}

		downloadDest := config.DesktopAppsPath()
		// Check disk utilisation by desktop directory
		usageBytes, err := file.DirSize(downloadDest)
		if err != nil {
			fatal.ExitErr(err, "Error checking ios build disk space usage")
		}
		log.Infof("Current desktop app build disk usage: %.2fGB", float64(usageBytes)/1024.0/1024.0/1024.0)

		appPath := appCmd.DownloadLatestApp(a, downloadDest)

		// Set env vars so they are available in the app process
		for k, v := range a.EnvVars {
			log.Debugf("Setting %s to %s", k, v)
			os.Setenv(k, v)
		}

		log.Info("☐ Launching app")

		// TODO probably want to figure out a better way to abstract opening an app cross platform
		if util.IsMacOS() {
			w := log.WithField("id", "tb-app-desktop-run-open").WriterLevel(log.DebugLevel)
			defer w.Close()
			err = command.New(command.WithStdout(w), command.WithStderr(w)).Exec("open", appPath)
		} else {
			fatal.Exit("tb app desktop run is not supported on your platform")
		}

		if err != nil {
			fatal.ExitErrf(err, "failed to run app %s", a.FullName())
		}

		log.Info("☑ Launched app")
		log.Info("🎉🎉🎉 Enjoy!")
	},
}

func init() {
	desktopCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&runOpts.branch, "branch", "b", "", "The name of the git branch associated build to pull down and run")
}
