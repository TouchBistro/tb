package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/TouchBistro/tb/util"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

var (
	defaultStopTimeout = 1 * time.Second
)

func ECRLogin() error {
	out, err := exec.Command("aws", strings.Fields("ecr get-login --region us-east-1 --no-include-email")...).Output()
	if err != nil {
		return errors.Wrap(err, "executing aws ecr get-login failed - try running aws configure.")
	}

	dockerLoginArgs := strings.Fields(string(out))
	err = util.Exec("ecr-login", dockerLoginArgs[0], dockerLoginArgs[1:]...)
	return errors.Wrap(err, "docker login failed")
}

func Pull(imageURI string) error {
	err := util.Exec(imageURI, "docker", "pull", imageURI)
	return err
}

func StopAllContainers() error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return errors.Wrap(err, "failed to create docker client")
	}

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true, Quiet: true})
	if err != nil {
		return errors.Wrap(err, "failed to list containers")
	}

	for _, container := range containers {
		if err := cli.ContainerStop(ctx, container.ID, &defaultStopTimeout); err != nil {
			return errors.Wrapf(err, "failed to stop container %s", container.ID)
		}
	}
	return nil
}

func RmContainers() error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return errors.Wrap(err, "failed to create docker client")
	}

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true, Quiet: true})
	if err != nil {
		return errors.Wrap(err, "failed to list containers")
	}

	for _, container := range containers {
		if err := cli.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{}); err != nil {
			return errors.Wrapf(err, "failed to remove container %s", container.ID)
		}
	}
	return nil
}

func RmImages() error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()

	if err != nil {
		return errors.Wrap(err, "failed to create docker client")
	}

	images, err := cli.ImageList(ctx, types.ImageListOptions{All: true})
	if err != nil {
		return errors.Wrap(err, "failed to list images")
	}

	for _, image := range images {
		if _, err := cli.ImageRemove(ctx, image.ID, types.ImageRemoveOptions{Force: true}); err != nil {
			return errors.Wrapf(err, "failed to remove image %s", image.ID)
		}
	}

	return nil
}

func PruneImages() (uint64, error) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return 0, errors.Wrap(err, "failed to create docker client")
	}
	args := filters.NewArgs()
	args.Add("dangling", "false")
	report, err := cli.ImagesPrune(ctx, args)
	if err != nil {
		return 0, errors.Wrap(err, "failed to prune images")
	}

	return report.SpaceReclaimed, nil
}

func RmNetworks() error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return errors.Wrap(err, "failed to create docker client")
	}

	networks, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list networks")
	}

	for _, network := range networks {
		if network.Name == "bridge" || network.Name == "host" || network.Name == "none" {
			continue
		}

		if err := cli.NetworkRemove(ctx, network.ID); err != nil {
			return errors.Wrapf(err, "failed to remove network %s", network.ID)
		}
	}

	return nil
}

func RmVolumes() error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return errors.Wrap(err, "failed to create docker client")
	}

	volumes, err := cli.VolumeList(ctx, filters.Args{})
	if err != nil {
		return errors.Wrap(err, "failed to list volumes")
	}

	for _, volume := range volumes.Volumes {
		if err := cli.VolumeRemove(ctx, volume.Name, true); err != nil {
			return errors.Wrapf(err, "failed to remove volume %s", volume.Name)
		}
	}

	return nil
}

func StopContainersAndServices(services string) error {
	var err error

	log.Debug("stopping compose services...")
	err = ComposeStop(services)
	if err != nil {
		return errors.Wrap(err, "failed stopping compose services")
	}
	log.Debug("...done")

	return nil
}

func CheckDockerDiskUsage() (bool, uint64, error) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return false, 0, errors.Wrap(err, "failed to create docker client")
	}

	disk, err := cli.DiskUsage(ctx)
	usage := uint64(disk.LayersSize)
	if err != nil {
		return false, 0, errors.Wrap(err, "could not retreive docker disk usage")
	}

	// TODO: This needs to be cleaned up
	dockerVmPath := fmt.Sprintf("%s/Library/Containers/com.docker.docker/Data/vms/0/data/Docker.raw", os.Getenv("HOME"))
	fs, err := os.Stat(dockerVmPath)
	if err != nil {
		// The location of the Docker.raw file was moved between docker releases, but only new installations are affected.
		// Here we check the original path oto support users with the old location.
		dockerVmPath = fmt.Sprintf("%s/Library/Containers/com.docker.docker/Data/vms/0/Docker.raw", os.Getenv("HOME"))
		fs, err = os.Stat(dockerVmPath)
		if err != nil {
			return false, usage, errors.Wrap(err, "could not retreive system disk usage")
		}
	}

	// if docker is using more than 60% our available docker space, probably cleanup
	if usage > uint64(float64(fs.Size())*0.6) {
		return true, usage, nil
	}
	return false, usage, nil
}
