package deps

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/TouchBistro/goutils/command"
	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/errkind"
	"github.com/TouchBistro/tb/util"
)

// dependency is an os dependency needed to run tb
type dependency struct {
	Name       string
	InstallCmd []string
}

// Dependency names because magic strings suck
const (
	Brew       = "brew"
	Lazydocker = "lazydocker"
	Mycli      = "mycli"
	Mssqlcli   = "mssql-cli"
	Pgcli      = "pgcli"
)

var deps = map[string]dependency{
	Brew: {
		Name:       "brew",
		InstallCmd: []string{"/bin/bash", "-c", "\"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install.sh)\""},
	},
	Pgcli: {
		Name:       "pgcli",
		InstallCmd: []string{"brew", "install", "pgcli"},
	},
	Mycli: {
		Name:       "mycli",
		InstallCmd: []string{"brew", "install", "mycli"},
	},
	Mssqlcli: {
		Name:       "mssql-cli",
		InstallCmd: []string{"pip", "install", "mssql-cli"},
	},
	Lazydocker: {
		Name:       "lazydocker",
		InstallCmd: []string{"brew", "install", "lazydocker"},
	},
}

func Resolve(ctx context.Context, depNames ...string) error {
	const op = errors.Op("deps.Resolve")
	tracker := progress.TrackerFromContext(ctx)
	tracker.Debugf("checking dependencies")
	if !util.IsMacOS() && !util.IsLinux() {
		return errors.New(
			errkind.Invalid,
			"tb currently supports Darwin (MacOS) and Linux only for installing dependencies",
			op,
		)
	}

	w := progress.LogWriter(tracker, tracker.WithFields(progress.Fields{"op": op}).Debug)
	defer w.Close()
	for _, depName := range depNames {
		dep, ok := deps[depName]
		if !ok {
			return errors.New(
				errkind.Internal,
				fmt.Sprintf("%s is not a valid dependency", depName),
				op,
			)
		}
		if command.IsAvailable(dep.Name) {
			tracker.Debugf("%s was found", dep.Name)
			continue
		}

		tracker.Warnf("%s was NOT found", dep.Name)
		tracker.Debugf("installing %s", dep.Name)
		cmd := exec.CommandContext(ctx, dep.InstallCmd[0], dep.InstallCmd[1:]...)
		cmd.Stdout = w
		cmd.Stderr = w
		if err := cmd.Run(); err != nil {
			return errors.Wrap(err, errors.Meta{
				Reason: fmt.Sprintf("install failed for %s", dep.Name),
				Op:     op,
			})
		}
		tracker.Debugf("finished installing %s.\n", dep.Name)
	}
	tracker.Debug("finished checking dependencies")
	return nil
}
