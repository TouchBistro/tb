package git

import (
	"fmt"

	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
)

func repoURL(repoName string) string {
	return fmt.Sprintf("git@github.com:TouchBistro/%s.git", repoName)
}

func Clone(repoName, destDir string) error {
	repoURL := repoURL(repoName)
	destPath := fmt.Sprintf("%s/%s", destDir, repoName)
	err := util.Exec(repoName, "git", "clone", repoURL, destPath)

	return errors.Wrapf(err, "exec failed to clone %s to %s", repoName, destDir)
}

func Pull(repoName, repoDir string) error {
	repoPath := fmt.Sprintf("%s/%s", repoDir, repoName)
	err := util.Exec(repoName, "git", "-C", repoPath, "pull")

	return errors.Wrapf(err, "exec failed to pull %s", repoName)
}
