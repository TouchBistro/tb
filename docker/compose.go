package docker

import (
	"fmt"
	"strings"

	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
)

const (
	stopTimeoutSecs = 2
)

func serviceString(services []string) (string, error) {
	var b strings.Builder
	for _, serviceName := range services {
		// Make sure it's a valid service
		// We should probably consider removing needing the config package here
		s, ok := config.Services()[serviceName]
		if !ok {
			msg := fmt.Sprintf("%s is not a valid service\n. Try running `tb list` to see available services\n", serviceName)
			return "", errors.New(msg)
		}

		b.WriteString(config.ComposeName(serviceName, s))
		b.WriteString(" ")
	}

	return b.String(), nil
}

func ComposeFile() string {
	return fmt.Sprintf("-f %s/docker-compose.yml", config.TBRootPath())
}

func ComposeStop(services []string) error {
	servicelist, err := serviceString(services)
	if err != nil {
		return errors.Wrap(err, "could not exec docker-compose stop")
	}
	stopArgs := fmt.Sprintf("%s stop -t %d %s", ComposeFile(), stopTimeoutSecs, servicelist)
	err = util.Exec("docker-compose-stop", "docker-compose", strings.Fields(stopArgs)...)

	return errors.Wrap(err, "could not exec docker-compose stop")
}

func ComposeRm(services []string) error {
	servicelist, err := serviceString(services)
	if err != nil {
		return errors.Wrap(err, "could not exec docker-compose rm")
	}
	rmArgs := fmt.Sprintf("%s rm -f %s", ComposeFile(), servicelist)
	err = util.Exec("docker-compose-stop", "docker-compose", strings.Fields(rmArgs)...)

	return errors.Wrap(err, "could not exec docker-compose rm")
}
