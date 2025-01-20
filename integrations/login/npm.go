package login

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/errkind"
)

type npmStrategy struct{}

func (npmStrategy) Name() string {
	return "NPM"
}

func (npmStrategy) Login(ctx context.Context) error {
	const op = errors.Op("login.npmStrategy.Login")
	const npmToken = "NPM_READ_TOKEN"
	tracker := progress.TrackerFromContext(ctx)
	tracker.Debugf("Checking if env var %s is set...", npmToken)
	if os.Getenv(npmToken) != "" {
		tracker.Debugf("Required env var %s is set", npmToken)
		return nil
	}

	homedir, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.Internal,
			Reason: "unable to find user home directory",
			Op:     op,
		})
	}
	npmrcPath := filepath.Join(homedir, ".npmrc")
	tracker.Debugf("Required env var %s not set\nChecking %s...", npmToken, npmrcPath)
	if !file.Exists(npmrcPath) {
		tracker.Warnf("%s not found.", npmrcPath)
		tracker.Warn("Log in to the npm registry with command: 'npm login' and try again.")
		return errors.New(errkind.Invalid, "not logged into npm registry", op)
	}

	tracker.Debugf("Looking for token in %s...", npmrcPath)
	data, err := os.ReadFile(npmrcPath)
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: fmt.Sprintf("failed to read %s", npmrcPath),
			Op:     op,
		})
	}

	regex := regexp.MustCompile(`//registry\.npmjs\.org/:_authToken\s*=\s*(.+)`)
	matches := regex.FindSubmatch(data)
	if matches == nil {
		tracker.Warnf("npm token not found in %s, make sure you are logged in", npmrcPath)
		return errors.New(errkind.Invalid, "no npm token", op)
	}

	tracker.Debugf("Found authToken. Setting env var %s...", npmToken)
	// Set the NPM_READ_TOKEN as an env var. This way any child processes run will inherit this env var
	// meaning when we run docker build it should have access to it
	// matches[0] is the full match
	os.Setenv(npmToken, string(matches[1]))
	return nil
}
