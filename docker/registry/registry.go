package registry

import (
	"time"

	"github.com/pkg/errors"
)

type ImageDetail struct {
	PushedAt *time.Time
	Tags     []string
}

type DockerRegistry interface {
	FetchRepoImages(ecrImage string, limit int) ([]ImageDetail, error)
}

func GetRegistry(registryName string) (DockerRegistry, error) {
	var provider DockerRegistry
	switch registryName {
	case "ecr":
		provider = ECRDockerRegistry{}
	default:
		return nil, errors.Errorf("Invalid docker registry %s", registryName)
	}

	return provider, nil
}
