package registry

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/tb/errkind"
)

type ImageDetail struct {
	PushedAt *time.Time
	Tags     []string
}

type DockerRegistry interface {
	FetchRepoImages(ctx context.Context, image string, limit int) ([]ImageDetail, error)
}

func Get(registryName string) (DockerRegistry, error) {
	switch strings.ToLower(registryName) {
	case "ecr":
		return ecrDockerRegistry{}, nil
	default:
		return nil, errors.New(
			errkind.Invalid,
			fmt.Sprintf("unknown docker registry %s", registryName),
			"registry.Get",
		)
	}
}
