package deps

import (
	"github.com/TouchBistro/goutils/command"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Dependency is an os dependency needed to run tb
type Dependency struct {
	Name          string
	InstallCmd    []string
	BeforeInstall func() error
	AfterInstall  func() error
}

// Dependency names because magic strings suck
const (
	Aws        = "aws"
	Brew       = "brew"
	Lazydocker = "lazydocker"
	Mycli      = "mycli"
	Mssqlcli   = "mssql-cli"
	Node       = "node"
	Pgcli      = "pgcli"
	Yarn       = "yarn"
)

var deps = map[string]Dependency{
	Brew: {
		Name:       "brew",
		InstallCmd: []string{"/bin/bash", "-c", "\"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install.sh)\""},
	},
	Pgcli: {
		Name: "pgcli",
		BeforeInstall: func() error {
			w := log.WithField("id", "pgcli-install").WriterLevel(log.DebugLevel)
			defer w.Close()
			cmd := command.New(command.WithStdout(w), command.WithStderr(w))
			err := cmd.Exec("brew", "tap", "dbcli/tap")
			return errors.Wrap(err, "failed to tap dbcli/tap")
		},
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
	Aws: {
		Name:       "aws",
		InstallCmd: []string{"brew", "install", "awscli"},
	},
	Lazydocker: {
		Name: "lazydocker",
		BeforeInstall: func() error {
			w := log.WithField("id", "lazydocker-install").WriterLevel(log.DebugLevel)
			defer w.Close()
			cmd := command.New(command.WithStdout(w), command.WithStderr(w))
			err := cmd.Exec("brew", "tap", "jesseduffield/lazydocker")
			return errors.Wrap(err, "failed to tap jesseduffield/lazydocker")
		},
		InstallCmd: []string{"brew", "install", "lazydocker"},
	},
	Node: {
		Name:       "node",
		InstallCmd: []string{"brew", "install", "node"},
	},
	Yarn: {
		Name:       "yarn",
		InstallCmd: []string{"brew", "install", "yarn"},
	},
}

func Resolve(depNames ...string) error {
	log.Info("☐ checking dependencies")

	if !util.IsMacOS() && !util.IsLinux() {
		fatal.Exit("tb currently supports Darwin (MacOS) and Linux only for installing dependencies. If you want to support other OSes, please make a pull request.\n")
	}

	for _, depName := range depNames {
		dep, ok := deps[depName]
		if !ok {
			return errors.Errorf("%s is not a valid dependency.", depName)
		}

		if command.IsAvailable(dep.Name) {
			log.Debugf("%s was found.\n", dep.Name)
			continue
		}

		log.Warnf("%s was NOT found.\n", dep.Name)
		log.Debugf("installing %s.\n", dep.Name)

		if dep.BeforeInstall != nil {
			err := dep.BeforeInstall()
			if err != nil {
				return errors.Wrapf(err, "before install failed for %s", dep.Name)
			}
		}

		installCmd := dep.InstallCmd[0]
		w := log.WithField("id", installCmd).WriterLevel(log.DebugLevel)
		defer w.Close()
		cmd := command.New(command.WithStdout(w), command.WithStderr(w))
		err := cmd.Exec(installCmd, dep.InstallCmd[1:]...)
		if err != nil {
			return errors.Wrapf(err, "install failed for %s", dep.Name)
		}

		if dep.AfterInstall != nil {
			err := dep.AfterInstall()
			if err != nil {
				return errors.Wrapf(err, "after install failed for %s", dep.Name)
			}
		}

		log.Debugf("finished installing %s.\n", dep.Name)
	}

	log.Info("☑ finished checking dependencies")
	return nil
}
