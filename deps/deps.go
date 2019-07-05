package deps

import (
	"os"
	"runtime"

	"github.com/TouchBistro/tb/util"
	log "github.com/sirupsen/logrus"
)

// Dependency is an os dependency needed to run core-devtools
type Dependency struct {
	Name          string
	InstallCmd    []string
	BeforeInstall func() error
	AfterInstall  func() error
}

var deps = []Dependency{
	// ROT IN HELL STEVE
	// Dependency{
	// 	Name:       "xcode-select -p",
	// 	InstallCmd: []string{"xcode-select", "--install"},
	// },
	Dependency{
		Name:       "brew",
		InstallCmd: []string{"/usr/bin/ruby", "-e", "\"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)\""},
	},
	Dependency{
		Name:       "pgcli",
		InstallCmd: []string{"brew", "install", "pgcli"},
	},
	Dependency{
		Name:       "jq",
		InstallCmd: []string{"brew", "install", "jq"},
	},
	Dependency{
		Name:       "aws",
		InstallCmd: []string{"brew", "install", "awscli"},
	},

	// Dependency{
	// 	Name:       "nvm",
	// 	InstallCmd: []string{"brew", "install", "nvm"},
	// 	AfterInstall: func() error {
	// 		home := os.Getenv("HOME") // TODO: Make portable for uzi?
	// 		dirPath := fmt.Sprintf("%s/.nvm", home)
	// 		if !util.FileOrDirExists(dirPath) {
	// 			err := os.Mkdir(dirPath, os.ModeDir)
	// 			return err
	// 		}
	// 		for _, rcFile := range []string{".zshrc", ".bashrc"} {
	// 			rcPath := fmt.Sprintf("%s/%s", home, rcFile)
	// 			fmt.Printf("...adding nvm export to %s\n", rcPath)
	// 			err := util.AppendLineToFile(rcPath, "export NVM_DIR=\"$HOME/.nvm\"")
	// 			err = util.AppendLineToFile(rcPath, ". \"/usr/local/opt/nvm/nvm.sh\"")
	// 			if err != nil {
	// 				return err
	// 			}
	// 		}
	// 		return nil
	// 	},
	// },

	// TODO: Check that `which node` resolves to something like /Users/<user>/.nvm/version/node/<version>/bin/node
	Dependency{
		Name:       "node",
		InstallCmd: []string{"nvm", "install", "stable"},
	},
	Dependency{
		Name:       "yarn",
		InstallCmd: []string{"brew", "install", "yarn"},
	},
	Dependency{
		Name: "docker",
		BeforeInstall: func() error {
			_, err := util.Exec("brew", "tap", "caskroom/versions")
			return err
		},
		InstallCmd: []string{"brew", "cask", "install", "docker"},
	},
}

func Resolve() error {
	log.Println("checking dependencies...")

	if runtime.GOOS != "darwin" {
		log.Println("tb currently supports Darwin (MacOS) only for installing dependencies.")
		log.Println("if you want to support other OSes, please make a pull request or tell Dev Acceleration.")
		os.Exit(1)
	}

	for _, dep := range deps {
		if util.IsCommandAvailable(dep.Name) {
			log.Printf("%s was found.\n", dep.Name)
			continue
		} else {
			log.Printf("%s was NOT found.\n", dep.Name)
		}

		log.Printf("installing %s.\n", dep.Name)

		if dep.BeforeInstall != nil {
			err := dep.BeforeInstall()
			if err != nil {
				return err
			}
		}

		installCmd := dep.InstallCmd[0]
		installArgs := dep.InstallCmd[1:]

		_, err := util.Exec(installCmd, installArgs...)
		if err != nil {
			return err
		}

		if dep.AfterInstall != nil {
			err := dep.AfterInstall()
			if err != nil {
				return err
			}
		}

		log.Printf("finished installing %s.\n", dep.Name)
	}
	return nil
}
