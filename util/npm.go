package util

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const npmToken = "NPM_TOKEN"

func NpmLogin() error {
	log.Infoln("Checking private npm repository token...")
	if os.Getenv(npmToken) != "" {
		log.Infof("Required env var %s is set\n", npmToken)
		return nil
	}

	log.Infof("Required env var %s not set\nChecking ~/.npmrc...\n", npmToken)

	npmrcPath := os.Getenv("HOME") + "/.npmrc"
	if !FileOrDirExists(npmrcPath) {
		log.Warnln("No ~/.npmrc found.")
		log.Warnln("Log in to the touchbistro npm registry with command: 'npm login' and try again.")
		log.Warnln("If this does not work...Create a https://www.npmjs.com/ account called: touchbistro-youremailname, then message DevOps to add you to the @touchbistro account")
		// TODO: We could also let them log in here and continue
		return errors.New("error not logged into npm registry")
	}

	log.Infoln("Looking for token in ~/.npmrc...")

	// figure this thing out
	token := "" // =$(tail -n1 ~/.npmrc | grep -o '//registry.npmjs.org/:_authToken=.*' | cut -f2 -d=)

	if token == "" {
		log.Warnln("could not parse authToken out of ~/.npmrc")
		return errors.New("error no npm token")
	}

	log.Infoln("Found authToken. adding to dotfiles and exporting")
	log.Infoln("...exporting NPM_TOKEN=$token")

	rcFiles := [...]string{".zshrc", ".bash_profile"}

	for _, file := range rcFiles {
		rcPath := fmt.Sprintf("%s/%s", os.Getenv("HOME"), file)
		log.Debugf("...adding export to %s.\n", rcPath)
		err := AppendLineToFile(rcPath, "export NPM_TOKEN="+token)
		if err != nil {
			return errors.Wrapf(err, "failed to export to file %s", file)
		}
		log.Debugln("...done")
	}

	log.Infoln("run 'source ~/.zshrc' or 'source ~/.bash_profile'")
	return errors.New("error source rc file")
}
