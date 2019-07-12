package docker

import (
	"fmt"
	"strings"

	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/util"
)

func ComposeFile() string {
	return fmt.Sprintf("-f %s/docker-compose.yml", config.TBRootPath())
}

func ComposeStop() error {
	stopArgs := fmt.Sprintf("%s stop", ComposeFile())
	err := util.Exec("docker-compose", strings.Fields(stopArgs)...)

	return err
}
