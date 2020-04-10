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
		InstallCmd: []string{"/usr/bin/ruby", "-e", "\"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)\""},
	},
	Pgcli: {
		Name: "pgcli",
		BeforeInstall: func() error {
			err := command.Exec("brew", []string{"tap", "dbcli/tap"}, "pgcli-install")
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
			err := command.Exec("brew", []string{"tap", "jesseduffield/lazydocker"}, "lazydocker-install")
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

func init() {
	// In the future maybe we could have a way to initialize deps based of the OS. This could allow for setting different install methods.
	// Using brew is fine for now though
	if util.IsLinux() {
		// Update brew install script if linux
		brew := deps[Brew]
		brew.InstallCmd = []string{"/bin/bash", "-c", "\"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install.sh)\""}
	}
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

		if command.IsCommandAvailable(dep.Name) {
			log.Debugf("%s was found.\n", dep.Name)
			continue
		} else {
			log.Warnf("%s was NOT found.\n", dep.Name)
		}

		log.Debugf("installing %s.\n", dep.Name)

		if dep.BeforeInstall != nil {
			err := dep.BeforeInstall()
			if err != nil {
				return errors.Wrapf(err, "before install failed for %s", dep.Name)
			}
		}

		installCmd := dep.InstallCmd[0]
		installArgs := dep.InstallCmd[1:]

		err := command.Exec(installCmd, installArgs, depName)
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
