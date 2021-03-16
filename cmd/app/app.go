package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/spinner"
	"github.com/TouchBistro/tb/app"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/git"
	"github.com/TouchBistro/tb/storage"
	"github.com/TouchBistro/tb/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "tb app allows running and managing different kinds of applications",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Put app specific configuration & setup logic here

		// Check if current command is an ios subcommand
		isIOSCommand := cmd.Parent().Name() == "ios"

		if isIOSCommand && !util.IsMacOS() {
			fatal.Exit("Error: tb app ios is only supported on macOS")
		}

		// Get global flag value
		noRegistryPull, err := cmd.Flags().GetBool("no-registry-pull")
		if err != nil {
			// This is a coding error
			fatal.ExitErr(err, "failed to get flag")
		}

		// Need to do this explicitly here since we are defining PersistentPreRun
		// PersistentPreRun overrides the parent command's one if defined, so the one in root won't be run.
		err = config.Init(config.InitOptions{
			UpdateRegistries: !noRegistryPull,
			LoadServices:     false,
			LoadApps:         true,
		})
		if err != nil {
			fatal.ExitErr(err, "Failed to initialize config files")
		}
	},
}

func AppCmd() *cobra.Command {
	return appCmd
}

func DownloadLatestApp(a app.App, downloadDest string) string {
	// Look up the latest build sha for user-specified branch and app.
	s3Dir := filepath.Join(a.Name, a.Branch)
	log.Debugf("Checking objects on %s in bucket %s matching prefix %s", a.Storage.Provider, a.Storage.Bucket, s3Dir)

	storageProvider, err := storage.GetProvider(a.Storage.Provider)
	if err != nil {
		fatal.ExitErrf(err, "Failed getting storage provider %s", a.Storage.Provider)
	}

	s3Builds, err := storageProvider.ListObjectKeysByPrefix(a.Storage.Bucket, s3Dir)
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

	localBranchDir := filepath.Join(downloadDest, a.FullName(), a.Branch)
	log.Debugf("Checking contents at %s to see if we need to download a new version from S3", localBranchDir)

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
		log.Debugf("Checking latest github sha for %s-%s", a.GitRepo, a.Branch)
		latestGitsha, err := git.GetBranchHeadSha(a.GitRepo, a.Branch)
		if err != nil {
			fatal.ExitErrf(err, "Failed getting branch head sha for %s-%s", a.GitRepo, a.Branch)
		}
		log.Debugf("Latest github sha is %s", latestGitsha)
		if !strings.HasPrefix(s3BuildFilename, latestGitsha) {
			log.Warnf("sha of s3 build %s does not match latest github sha %s for branch %s", s3BuildFilename, latestGitsha, a.Branch)
		}

		currentSha := strings.Split(filepath.Base(localBuild), ".")[0]
		s3Sha := strings.Split(s3BuildFilename, ".")[0]

		log.Debugf("Current local build sha is %s", currentSha)
		log.Debugf("Latest s3 sha is %s", s3Sha)

		if currentSha == s3Sha {
			log.Debug("Current build sha matches remote sha")
		} else {
			log.Debug("Current build sha is different from s3 sha. Deleting local version...")
			err := os.RemoveAll(localBranchDir)
			if err != nil {
				fatal.ExitErrf(err, "failed to delete %s", localBranchDir)
			}
			refreshLocalBuild = true
		}
	}

	// Path where the downloaded app is
	dstPath := filepath.Join(downloadDest, a.FullName(), a.Branch, s3BuildFilename)

	// If there are no local builds or if our local build was deemed out of date, download the latest object from S3
	if len(localBuilds) == 0 || refreshLocalBuild {
		log.Debugf("Downloading %s from bucket %s to %s", pathToS3Tarball, a.Storage.Bucket, downloadDest)
		s := spinner.New(
			spinner.WithStartMessage("Downloading app "+a.FullName()),
			spinner.WithStopMessage("Finished downloading app "+a.FullName()),
			spinner.WithPersistMessages(log.IsLevelEnabled(log.DebugLevel)),
		)
		log.SetOutput(s)
		s.Start()

		err = storageProvider.DownloadObject(a.Storage.Bucket, pathToS3Tarball, dstPath)
		if err != nil {
			s.Stop()
			fatal.ExitErrf(err, "Failed to download a file from s3 from %s to %s", pathToS3Tarball, downloadDest)
		}

		// Untar, ungzip and cleanup the file
		log.Debugf("untar-ing %s", dstPath)
		err := util.Untar(dstPath, true)
		s.Stop()
		if err != nil {
			fatal.ExitErrf(err, "Failed to untar or cleanup app archive at %s", dstPath)
		}
		log.SetOutput(os.Stderr)
	}

	return strings.TrimSuffix(dstPath, ".tgz")
}
