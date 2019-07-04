package docker

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/TouchBistro/tb/src/util"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

func ECRLogin() error {
	out, err := exec.Command("aws", strings.Fields("ecr get-login --region us-east-1 --no-include-email")...).Output()
	if err != nil {
		return err
	}

	dockerLoginArgs := strings.Fields(string(out))
	err = util.Exec(dockerLoginArgs[0], dockerLoginArgs[1:]...)
	return nil
}

func Pull(imageURI string) error {
	err := util.Exec("docker", "pull", imageURI)
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
		fmt.Println(container.ID)
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
