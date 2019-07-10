package docker

import (
	"os/exec"
	"strings"

	"github.com/TouchBistro/tb/util"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

func ECRLogin() error {
	out, err := exec.Command("aws", strings.Fields("ecr get-login --region us-east-1 --no-include-email")...).Output()
	if err != nil {
		return err
	}

	dockerLoginArgs := strings.Fields(string(out))
	err = util.Exec(dockerLoginArgs[0], dockerLoginArgs[1:]...)
	return err
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
			log.WithFields(log.Fields{"error": err.Error(), "containerID": container.ID}).Debug("Failed to remove container")
			return err
		}
	}
	return nil
}

func RmImages() error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()

	if err != nil {
		return err
	}

	images, err := cli.ImageList(ctx, types.ImageListOptions{All: true})
	if err != nil {
		return err
	}

	for _, image := range images {
		if _, err := cli.ImageRemove(ctx, image.ID, types.ImageRemoveOptions{}); err != nil {
			log.WithFields(log.Fields{"error": err.Error(), "ImageID": image.ID}).Debug("Failed to remove image")
			return err
		}
	}

	return nil
}

func RmNetworks() error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()

	if err != nil {
		return err
	}

	networks, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return err
	}

	for _, network := range networks {
		if network.Name == "bridge" || network.Name == "host" || network.Name == "none" {
			continue
		}

		if err := cli.NetworkRemove(ctx, network.ID); err != nil {
			log.WithFields(log.Fields{"error": err.Error(), "NetworkID": network.ID}).Debug("Failed to remove network.")
			return err
		}
	}

	return nil
}

func RmVolumes() error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()

	if err != nil {
		return err
	}

	volumes, err := cli.VolumeList(ctx, filters.Args{})
	if err != nil {
		return err
	}

	for _, volume := range volumes.Volumes {
		if err := cli.VolumeRemove(ctx, volume.Name, true); err != nil {
			log.WithFields(log.Fields{"error": err.Error(), "VolumeID": volume.Name}).Debug("Failed to remove volume.")
			return err
		}
	}

	return nil
}

func StopContainersAndServices() error {
	var err error

	log.Debug("stopping running containers...")
	err = StopAllContainers()
	if err != nil {
		return err
	}
	log.Debug("...done")

	log.Debug("stopping compose services...")
	err = ComposeStop()
	if err != nil {
		return err
	}
	log.Debug("...done")

	return nil
}
