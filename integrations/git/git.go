// Package git provides functionality for working with Git repositories.
package git

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/errkind"
)

// Git is the interface that represents supported Git functionality.
type Git interface {
	Clone(ctx context.Context, repo, path string) error
	Pull(ctx context.Context, path string) error
	GetBranchHeadSha(ctx context.Context, repo, branch string) (string, error)
}

type realGit struct{}

func New() Git {
	return realGit{}
}

func (realGit) Clone(ctx context.Context, repo, path string) error {
	// TODO(@cszatmary): Should have support for HTTPS too.
	// Maybe we could check if it's a full URL or just the repo name?
	// If just repo name then assume SSH.
	repoURL := fmt.Sprintf("git@github.com:%s.git", repo)
	return execGit(ctx, "git.Git.Clone", nil, "clone", repoURL, path)
}

func (realGit) Pull(ctx context.Context, path string) error {
	return execGit(ctx, "git.Git.Pull", nil, "-C", path, "pull")
}

func (realGit) GetBranchHeadSha(ctx context.Context, repo, branch string) (string, error) {
	const op = errors.Op("git.Git.GetBranchHeadSha")
	repoURL := fmt.Sprintf("git@github.com:%s.git", repo)
	var stdout bytes.Buffer
	err := execGit(ctx, op, &stdout, "ls-remote", repoURL, branch)
	if err != nil {
		return "", err
	}
	result := stdout.String()
	if len(result) < 40 {
		return "", errors.New(
			errkind.Git,
			fmt.Sprintf("ls-remote sha too short from %s of %s", branch, repo),
			op,
		)
	}
	return result[0:40], nil
}

func execGit(ctx context.Context, op errors.Op, stdout io.Writer, args ...string) error {
	tracker := progress.TrackerFromContext(ctx)
	w := progress.LogWriter(tracker, tracker.WithFields(progress.Fields{"op": op}).Debug)
	defer w.Close()
	if stdout == nil {
		stdout = w
	}

	finalArgs := append([]string{"git"}, args...)
	cmd := exec.CommandContext(ctx, finalArgs[0], finalArgs[1:]...)
	cmd.Env = os.Environ()
	// Disable prompting for passwords via ssh, fail fast instead.
	// ref: https://groups.google.com/g/golang-codereviews/c/yOfVktgHf3M?pli=1
	cmd.Env = append(cmd.Env, "GIT_SSH_COMMAND=ssh -o BatchMode=yes")
	cmd.Stdout = stdout
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.Git,
			Reason: fmt.Sprintf("failed to run %q", strings.Join(finalArgs, " ")),
			Op:     op,
		})
	}
	return nil
}
