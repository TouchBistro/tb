package git

import (
	"fmt"

	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
)

func repoURL(repoName string) string {
	return fmt.Sprintf("git@github.com:TouchBistro/%s.git", repoName)
}

func RClone(success chan string, failed chan error, repoName, destDir string) {
	repoURL := repoURL(repoName)
	destPath := fmt.Sprintf("%s/%s", destDir, repoName)
	err := util.Exec("git", "clone", "--depth", "1", repoURL, destPath)
	if err != nil {
		failed <- err
	} else {
		success <- repoName
	}
}

func Pull(repoName, repoDir string) error {
	repoPath := fmt.Sprintf("%s/%s", repoDir, repoName)
	err := util.Exec("git", "-C", repoPath, "pull")

	return errors.Wrapf(err, "exec failed to pull %s", repoName)
}

func RPull(success chan string, failed chan error, repoName, repoDir string) {
	repoPath := fmt.Sprintf("%s/%s", repoDir, repoName)
	err := util.Exec("git", "-C", repoPath, "pull")
	if err != nil {
		failed <- errors.Wrapf(err, "exec failed to pull %s", repoName)
	} else {
		success <- repoName
	}
}
