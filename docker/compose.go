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

func serviceString(services []string) string {
	var b strings.Builder
	for _, serviceName := range services {
		// Make sure it's a valid service
		s, ok := config.Services()[serviceName]
		if !ok {
			fatal.Exitf("%s is not a valid service\n. Try running `tb list` to see available services\n", serviceName)
		}

		b.WriteString(config.ComposeName(serviceName, s))
		b.WriteString(" ")
	}

	return b.String()
}

func ComposeFile() string {
	return fmt.Sprintf("-f %s/docker-compose.yml", config.TBRootPath())
}

func ComposeStop(services []string) error {
	stopArgs := fmt.Sprintf("%s stop -t %d %s", ComposeFile(), stopTimeoutSecs, serviceString(services))
	err := util.Exec("docker-compose-stop", "docker-compose", strings.Fields(stopArgs)...)

	return errors.Wrap(err, "could not exec docker-compose stop")
}

func ComposeRm(services []string) error {
	rmArgs := fmt.Sprintf("%s rm -f %s", ComposeFile(), serviceString(services))
	err := util.Exec("docker-compose-stop", "docker-compose", strings.Fields(rmArgs)...)

	return errors.Wrap(err, "could not exec docker-compose rm")
}
