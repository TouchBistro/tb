package git

import (
	"bytes"
	"fmt"
	"io"
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

func GetBranchHeadSha(repo, branch string) (string, error) {
	repoUrl := "git@github.com:" + repo + ".git"
	w := log.WithField("id", "git-ls-remote").WriterLevel(log.DebugLevel)
	defer w.Close()
	stdout := new(bytes.Buffer)
	mw := io.MultiWriter(stdout, w)
	cmd := command.New(command.WithStdout(mw), command.WithStderr(w))
	err := cmd.Exec("git", "ls-remote", repoUrl, branch)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get %s head sha of %s", branch, repo)
	}

	result := stdout.String()
	if len(result) < 40 {
		return "", errors.Errorf("ls-remote sha too short from %s of %s", branch, repo)
	}
	return result[0:40], nil
}

func execGit(id string, args ...string) error {
	w := log.WithField("id", id).WriterLevel(log.DebugLevel)
	defer w.Close()
	cmd := command.New(command.WithStdout(w), command.WithStderr(w))
	return cmd.Exec("git", args...)
}
