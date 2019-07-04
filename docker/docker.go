package docker

import (
	"fmt"
	"github.com/TouchBistro/tb/util"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

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
		fmt.Println(container.ID)
		if err := cli.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{}); err != nil {
			return err
		}
	}
	return nil
}
