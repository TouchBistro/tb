// Package git provides functionality for working with Git repositories.
package git

import (
	"context"
	"fmt"
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
	return execGit(ctx, "git.Git.Clone", "git-clone", "clone", repoURL, path)
}

func (realGit) Pull(ctx context.Context, path string) error {
	return execGit(ctx, "git.Git.Pull", "git-pull", "-C", path, "pull")
}

func execGit(ctx context.Context, op errors.Op, id string, args ...string) error {
	tracker := progress.TrackerFromContext(ctx)
	w := progress.LogWriter(tracker, tracker.WithFields(progress.Fields{"id": id}).Debug)
	defer w.Close()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.Git,
			Reason: fmt.Sprintf("failed to run %q", strings.Join(args, " ")),
			Op:     op,
		})
	}
	return nil
}
