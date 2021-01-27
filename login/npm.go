package login

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/TouchBistro/goutils/file"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const npmToken = "NPM_TOKEN"

type NPMLoginStrategy struct{}

func (s NPMLoginStrategy) Name() string {
	return "NPM"
}

func (s NPMLoginStrategy) Login() error {
	log.Debugf("Checking if env var %s is set...", npmToken)
	if os.Getenv(npmToken) != "" {
		log.Debugf("Required env var %s is set", npmToken)
		return nil
	}

	npmrcPath := filepath.Join(os.Getenv("HOME"), ".npmrc")
	log.Debugf("Required env var %s not set\nChecking %s...", npmToken, npmrcPath)

	if !file.Exists(npmrcPath) {
		log.Warnf("%s not found.", npmrcPath)
		log.Warnln("Log in to the npm registry with command: 'npm login' and try again.")
		// TODO: We could also let them log in here and continue
		return errors.New("error not logged into npm registry")
	}

	log.Debugf("Looking for token in %s...", npmrcPath)

	data, err := ioutil.ReadFile(npmrcPath)
	if err != nil {
		return errors.Wrapf(err, "failed to read %s", npmrcPath)
	}

	regex := regexp.MustCompile(`//registry\.npmjs\.org/:_authToken\s*=\s*(.+)`)
	matches := regex.FindStringSubmatch(string(data))
	if matches == nil {
		log.Warnf("npm token not found in %s, make sure you are logged in", npmrcPath)
		return errors.New("error no npm token")
	}

	// matches[0] is the full match
	token := matches[1]

	log.Debugf("Found authToken. Setting env var %s...", npmToken)

	// Set the NPM_TOKEN as an env var. This way any child processes run will inherit this env var
	// meaning when we run docker build it should have access to it
	os.Setenv(npmToken, token)
	return nil
}
