package deps

import (
	"runtime"

	"github.com/TouchBistro/tb/fatal"
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
	// XcodeSelect = "xcode-select"
	Brew       = "brew"
	Pgcli      = "pgcli"
	Aws        = "aws"
	Lazydocker = "lazydocker"
	Node       = "node"
	Yarn       = "yarn"
)

var deps = map[string]Dependency{
	// ROT IN HELL STEVE
	// XcodeSelect: Dependency{
	// 	Name:       "xcode-select -p",
	// 	InstallCmd: []string{"xcode-select", "--install"},
	// },
	Brew: {
		Name:       "brew",
		InstallCmd: []string{"/usr/bin/ruby", "-e", "\"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)\""},
	},
	Pgcli: {
		Name: "pgcli",
		BeforeInstall: func() error {
			err := util.Exec("brew", "tap", "dbcli/tap")
			return errors.Wrap(err, "failed to tap dbcli/tap")
		},
		InstallCmd: []string{"brew", "install", "pgcli"},
	},
	Aws: {
		Name:       "aws",
		InstallCmd: []string{"brew", "install", "awscli"},
	},
	Lazydocker: {
		Name: "lazydocker",
		BeforeInstall: func() error {
			err := util.Exec("brew", "tap", "jesseduffield/lazydocker")
			return errors.Wrap(err, "failed to tap jesseduffield/lazydocker")
		},
		InstallCmd: []string{"brew", "install", "lazydocker"},
	},
	// TODO: Check that `which node` resolves to something like /Users/<user>/.nvm/version/node/<version>/bin/node
	Node: {
		Name:       "node",
		InstallCmd: []string{"nvm", "install", "stable"},
	},
	Yarn: {
		Name:       "yarn",
		InstallCmd: []string{"brew", "install", "yarn"},
	},
}

func Resolve(depNames ...string) error {
	log.Info("☐ checking dependencies")

	if runtime.GOOS != "darwin" {
		fatal.Exit("tb currently supports Darwin (MacOS) only for installing dependencies. if you want to support other OSes, please make a pull request or tell Dev Acceleration.\n")
	}

	for _, depName := range depNames {
		dep, ok := deps[depName]

		if !ok {
			return errors.Errorf("%s is not a valid dependency.", depName)
		}

		if util.IsCommandAvailable(dep.Name) {
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

		err := util.Exec(installCmd, installArgs...)
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
