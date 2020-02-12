package git

import (
	"fmt"
	"path/filepath"

	"github.com/TouchBistro/goutils/command"
	"github.com/pkg/errors"
)

func Clone(repo, destPath string) error {
	repoURL := fmt.Sprintf("git@github.com:%s.git", repo)
	err := command.Exec("git", []string{"clone", repoURL, destPath}, repo)

	return errors.Wrapf(err, "exec failed to clone %s to %s", repo, destPath)
}

func Pull(repo, repoDir string) error {
	repoPath := filepath.Join(repoDir, repo)
	err := command.Exec("git", []string{"-C", repoPath, "pull"}, repo)

	return errors.Wrapf(err, "exec failed to pull %s", repo)
}
