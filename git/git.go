package git

import (
	"fmt"
	"path/filepath"

	"github.com/TouchBistro/goutils/command"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func Clone(repo, destPath string) error {
	repoURL := fmt.Sprintf("git@github.com:%s.git", repo)
	err := execGit("git-clone", "clone", repoURL, destPath)
	if err != nil {
		return errors.Wrapf(err, "failed to clone %s to %s", repo, destPath)
	}
	return nil
}

func Pull(repo, repoDir string) error {
	repoPath := filepath.Join(repoDir, repo)
	err := execGit("git-pull", "-C", repoPath, "pull")
	if err != nil {
		return errors.Wrapf(err, "failed to pull %s", repo)
	}
	return nil
}

func execGit(id string, args ...string) error {
	w := log.WithField("id", id).WriterLevel(log.DebugLevel)
	defer w.Close()
	cmd := command.New(command.WithStdout(w), command.WithStderr(w))
	return cmd.Exec("git", args...)
}
