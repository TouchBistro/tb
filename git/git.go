package git

import (
	"fmt"
	"path/filepath"

	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
)

func Clone(repo, destPath string) error {
	repoURL := fmt.Sprintf("git@github.com:%s.git", repo)
	err := util.Exec(repo, "git", "clone", repoURL, destPath)

	return errors.Wrapf(err, "exec failed to clone %s to %s", repo, destPath)
}

func Pull(repo, repoDir string) error {
	repoPath := filepath.Join(repoDir, repo)
	err := util.Exec(repo, "git", "-C", repoPath, "pull")

	return errors.Wrapf(err, "exec failed to pull %s", repo)
}
