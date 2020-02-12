package login

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const npmToken = "NPM_TOKEN"

type NPMLoginStrategy struct{}

func (s NPMLoginStrategy) Name() string {
	return "NPM"
}

func (s NPMLoginStrategy) Login() error {
	log.Debugln("Checking private npm repository token...")
	if os.Getenv(npmToken) != "" {
		log.Debugf("Required env var %s is set\n", npmToken)
		return nil
	}

	log.Debugf("Required env var %s not set\nChecking ~/.npmrc...\n", npmToken)

	npmrcPath := os.Getenv("HOME") + "/.npmrc"
	if !util.FileOrDirExists(npmrcPath) {
		log.Warnln("No ~/.npmrc found.")
		log.Warnln("Log in to the touchbistro npm registry with command: 'npm login' and try again.")
		log.Warnln("If this does not work...Create a https://www.npmjs.com/ account called: touchbistro-youremailname, then message DevOps to add you to the @touchbistro account")
		// TODO: We could also let them log in here and continue
		return errors.New("error not logged into npm registry")
	}

	log.Debugln("Looking for token in ~/.npmrc...")

	// Do lazy way for now, npmrc usually is pretty small anyway
	data, err := ioutil.ReadFile(npmrcPath)
	if err != nil {
		return errors.Wrap(err, "failed to read ~/.npmrc")
	}

	r, err := regexp.Compile("//registry.npmjs.org/:_authToken=(.*)")
	if err != nil {
		return errors.Wrap(err, "unable to compile token regex")
	}

	token := r.FindStringSubmatch(string(data))[1]
	if token == "" {
		log.Warnln("could not parse authToken out of ~/.npmrc")
		return errors.New("error no npm token")
	}

	log.Debugln("Found authToken. adding to dotfiles and exporting")
	log.Debugln("...exporting NPM_TOKEN=$token")

	rcFiles := [...]string{".zshrc", ".bash_profile"}

	r, err = regexp.Compile("export NPM_TOKEN=.+")
	if err != nil {
		return errors.Wrap(err, "unable to compile export regex")
	}

	for _, file := range rcFiles {
		rcPath := filepath.Join(os.Getenv("HOME"), file)
		if !util.FileOrDirExists(rcPath) {
			log.Debugf("No %s, skipping", file)
			continue
		}

		contents, err := ioutil.ReadFile(rcPath)
		if err != nil {
			return errors.Wrapf(err, "failed to read ~/%s", file)
		}

		export := r.FindString(string(contents))
		if export != "" {
			log.Debugf("Export found in ~/%s", file)
			continue
		}

		log.Debugf("...adding export to %s.\n", rcPath)
		err = util.AppendLineToFile(rcPath, "export NPM_TOKEN="+token)
		if err != nil {
			return errors.Wrapf(err, "failed to export to file %s", file)
		}
		log.Debugln("...done")
	}

	log.Debugln("run 'source ~/.zshrc' or 'source ~/.bash_profile'")
	return errors.New("error source rc file")
}
