package docker

import (
	"strings"

	"github.com/TouchBistro/tb/util"
	"os/exec"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

func ComposeFiles() (string, error) {
	matches, err := filepath.Glob("./docker-compose.*.yml")

	if err != nil {
		return "", err
	}

	if len(matches) == 0 {
		return "", nil
	}

	str := "-f " + strings.Join(matches, " -f ")

	return str, nil
}

func ECRLogin() error {
	out, err := exec.Command("aws", strings.Fields("ecr get-login --region us-east-1 --no-include-email")...).Output()
	if err != nil {
		return err
	}

	dockerLoginArgs := strings.Fields(string(out))
	_, err = util.Exec(dockerLoginArgs[0], dockerLoginArgs[1:]...)
	return err
}

func Pull(imageURI string) error {
	_, err := util.Exec("docker", "pull", imageURI)
	return err
}

func StopAllContainers() error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true, Quiet: true})
	if err != nil {
		return err
	}

	for _, container := range containers {
		if err := cli.ContainerStop(ctx, container.ID, nil); err != nil {
			return err
		}
	}
	return nil
}

func RmContainers() error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true, Quiet: true})
	if err != nil {
		return err
	}

	for _, container := range containers {
		if err := cli.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{}); err != nil {
			return err
		}
	}
	return nil
}
