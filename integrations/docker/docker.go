package docker

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/errkind"
	"github.com/docker/docker/client"
)

// Docker represents functionality provided by docker.
type Docker interface {
	PullImage(ctx context.Context, imageURI string) error
}

// New returns a new Docker instance.
func New() (Docker, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, errors.Wrap(err, errors.Meta{
			Kind:   errkind.Docker,
			Reason: "unable to create docker client",
			Op:     "docker.New",
		})
	}
	return docker{cli}, nil
}

type docker struct {
	client *client.Client
}

func (d docker) PullImage(ctx context.Context, imageURI string) error {
	tracker := progress.TrackerFromContext(ctx)
	w := progress.LogWriter(tracker, tracker.WithFields(progress.Fields{"id": "docker-pull"}).Debug)
	defer w.Close()
	cmd := exec.CommandContext(ctx, "docker", "pull", imageURI)
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.Docker,
			Reason: fmt.Sprintf("failed to pull docker image %s", imageURI),
			Op:     "docker.Docker.ImagePull",
		})
	}

	// TODO(@cszatmary): The docker SDK supports pulling images but it requires explicit auth.
	// I'd like to switch to using that, but it will require some changes to how we handle logins.
	// rc, err := d.client.ImagePull(ctx, imageURI, types.ImagePullOptions{})
	// if err != nil {
	// 	return errors.New(errors.Docker, fmt.Sprintf("failed to pull docker image %s", imageURI), op, err)
	// }
	// defer rc.Close()
	return nil
}
