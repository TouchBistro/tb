package ios

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TouchBistro/tb/awss3"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/fatal"
	"github.com/TouchBistro/tb/git"
	"github.com/TouchBistro/tb/simulator"
	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	iosVersion string
	deviceName string
	dataPath   string
	appName    string
	branch     string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Runs an iOS app build in an iOS Simulator",
	Args:  cobra.ExactArgs(0),
	Long: `Runs an iOS app build in an iOS Simulator.

Examples:
- run the current master build in the default iOS Simulator
	tb ios run

- run the build for specific branch in an iOS 12.3 iPad Air 2 simulator
	tb ios run --app TouchBistro --ios-version 12.3 --device iPad Air 2 --branch task/pay-631/fix-thing`,
	Run: func(cmd *cobra.Command, args []string) {
		app, ok := config.Apps()[appName]
		if !ok {
			fatal.Exitf("Error: No iOS app with name %s\n", appName)
		}

		downloadDest := config.IOSBuildPath()

		// Check disk utilisation by ios directory
		usageBytes, err := util.DirSize(downloadDest)
		if err != nil {
			fatal.ExitErr(err, "Error checking ios build disk space usage")
		}
		log.Infof("Current ios build disk usage: %.2fGB", float64(usageBytes)/1024.0/1024.0/1024.0)

		// Look up the latest build sha for user-specified branch and app.
		s3Dir := fmt.Sprintf("%s/%s", appName, branch)
		log.Infof("Checking objects on aws in bucket %s matching prefix %s...", config.Bucket, s3Dir)
		s3Builds, err := awss3.ListObjectKeysByPrefix(config.Bucket, s3Dir)
		if err != nil {
			fatal.ExitErrf(err, "Failed getting keys from s3 in dir %s", s3Dir)
		}
		if len(s3Builds) == 0 {
			fatal.Exitf("could not find any builds for %s", s3Dir)
		} else if len(s3Builds) > 1 {
			// We only expect one build per branch. If we find two, its likely a bug or some kind of
			// race condition from the build-uploading side.
			// If this gets clunky we can determine a sort order for the builds.
			fatal.Exitf("Got the following builds for this branch %+v. Only expecting one build", s3Builds)
		}

		pathToS3Tarball := s3Builds[0]
		s3BuildFilename := filepath.Base(pathToS3Tarball)

		// Decide whether or not to pull down a new version.

		localBranchDir := filepath.Join(downloadDest, appName, branch)
		log.Infof("Checking contents at %s to see if we need to download a new version from S3", localBranchDir)

		pattern := fmt.Sprintf("%s/*.app", localBranchDir)
		localBuilds, err := filepath.Glob(pattern)
		if err != nil {
			fatal.ExitErrf(err, "couldn't glob for %s", pattern)
		}

		if len(localBuilds) > 1 {
			fatal.Exitf("Got the following builds: %+v. Only expecting one build", localBuilds)
		}

		// If there is a local build, compare its sha against s3 and github versions
		var refreshLocalBuild bool
		if len(localBuilds) == 1 {
			localBuild := localBuilds[0]

			// If there is a local build, get latest sha from github for desired branch to see if the build available on s3 corresponds to the
			// latest commit on the branch.
			log.Infof("Checking latest github sha for %s/%s-%s", app.Organisation, app.Repo, branch)
			latestGitsha, err := git.GetBranchHeadSha(app.Organisation, app.Repo, branch)
			if err != nil {
				fatal.ExitErrf(err, "Failed getting branch head sha for %s/%s", app.Repo, branch)
			}
			log.Infof("Latest github sha is %s", latestGitsha)
			if !strings.HasPrefix(s3BuildFilename, latestGitsha) {
				log.Warnf("sha of s3 build %s does not match latest github sha %s for branch %s", s3BuildFilename, latestGitsha, branch)
			}

			currentSha := strings.Split(filepath.Base(localBuild), ".")[0]
			s3Sha := strings.Split(s3BuildFilename, ".")[0]

			log.Infof("Current local build sha is %s", currentSha)
			log.Infof("Latest s3 sha is %s", s3Sha)

			if currentSha == s3Sha {
				log.Infoln("Current build sha matches remote sha")
			} else {
				log.Infoln("Current build shais different from s3 sha. Deleting local version...")
				err := os.RemoveAll(localBranchDir)
				if err != nil {
					fatal.ExitErrf(err, "failed to delete %s", localBranchDir)
				}

				refreshLocalBuild = true
			}
		}

		// If there are no local builds or if our local build was deemed out of date, download the latest object from S3
		if len(localBuilds) == 0 || refreshLocalBuild {
			log.Infof("Downloading %s from bucket %s to %s", pathToS3Tarball, config.Bucket, downloadDest)
			successCh := make(chan string)
			failedCh := make(chan error)
			go func(successCh chan string, failedCh chan error) {
				err = awss3.DownloadObject(config.Bucket, pathToS3Tarball, downloadDest)
				if err != nil {
					failedCh <- errors.Wrapf(err, "Failed to download a file from s3 from %s to %s", pathToS3Tarball, downloadDest)
					return
				}
				successCh <- pathToS3Tarball
			}(successCh, failedCh)
			count := 1
			util.SpinnerWait(successCh, failedCh, "\t☑ finished downloading %s\n", "failed S3 download", count)

			// Untar, ungzip and cleanup the file
			pathToLocalTarball := filepath.Join(downloadDest, pathToS3Tarball)
			log.Infof("untar-ing %s", pathToLocalTarball)
			err := util.Untar(pathToLocalTarball, true)
			if err != nil {
				fatal.ExitErrf(err, "Failed to untar or cleanup app archive at %s", pathToLocalTarball)
			}
		}

		appPath := filepath.Join(downloadDest, strings.TrimSuffix(pathToS3Tarball, ".tgz"))

		log.Debugln("☐ Finding device UDID")
		deviceUDID, err := simulator.GetDeviceUDID("iOS "+iosVersion, deviceName)
		if err != nil {
			fatal.ExitErr(err, "☒ Failed to get device UDID.\nRun \"xcrun simctl list devices\" to list available simulators.")
		}

		log.Debugf("☑ Found device UDID: %s\n", deviceUDID)
		log.Infof("☐ Booting Simulator %s\n", deviceName)

		err = simulator.Boot(deviceUDID)
		if err != nil {
			fatal.ExitErrf(err, "☒ Failed to boot simulator %s", deviceName)
		}

		log.Infof("☑ Booted simulator %s\n", deviceName)
		log.Debugln("☐ Opening simulator app")

		err = simulator.Open()
		if err != nil {
			fatal.ExitErr(err, "☒ Failed to launch simulator")
		}

		log.Debugln("☑ Opened simulator app")
		log.Infof("☐ Installing app on %s\n", deviceName)

		err = simulator.InstallApp(deviceUDID, appPath)
		if err != nil {
			fatal.ExitErrf(err, "☒ Failed to install app at path %s on simulator %s", appPath, deviceName)
		}

		log.Infof("☑ Installed app %s on %s\n", app.BundleID, deviceName)

		if dataPath != "" {
			log.Infoln("☐ Injecting data files into simulator")

			appDataPath, err := simulator.GetAppDataPath(deviceUDID, app.BundleID)
			if err != nil {
				fatal.ExitErrf(err, "Failed to get path to data for app %s", app.BundleID)
			}

			err = util.CopyDirContents(dataPath, appDataPath)
			if err != nil {
				fatal.ExitErrf(err, "☒ Failed to inject data into simulator")
			}

			log.Infoln("☑ Injected data into simulator")
		}

		log.Info("☐ Launching app in simulator")

		err = simulator.LaunchApp(deviceUDID, app.BundleID)
		if err != nil {
			fatal.ExitErrf(err, "☒ Failed to launch app %s on simulator %s", app.BundleID, deviceName)
		}

		log.Infof("☑ Launched app %s on %s\n", app.BundleID, deviceName)
		log.Info("🎉🎉🎉 Enjoy!")
	},
}

func init() {
	iosCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&iosVersion, "ios-version", "i", "13.1", "The iOS version to use")
	runCmd.Flags().StringVarP(&deviceName, "device", "d", "iPad Air (3rd generation)", "The name of the device to use")
	runCmd.Flags().StringVarP(&appName, "app", "a", "TouchBistro", "The name of the application to run, eg TouchBistro")
	runCmd.Flags().StringVarP(&branch, "branch", "b", "master", "The name of the git branch associated build to pull down and run")
	runCmd.Flags().StringVarP(&dataPath, "data-path", "D", "", "The path to a data directory to inject into the simulator")
}
