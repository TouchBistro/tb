package config

import "fmt"

type Service struct {
	IsGithubRepo bool   `yaml:"repo"`
	Migrations   bool   `yaml:"migrations"`
	ECR          bool   `yaml:"ecr"`
	ECRTag       string `yaml:"ecrTag"`
}

type ServiceOverride struct {
	ECR    bool   `yaml:"ecr"`
	ECRTag string `yaml:"ecrTag"`
}

type ServiceMap = map[string]Service

func ResolveEcrURI(service, tag string) string {
	return fmt.Sprintf("%s/%s:%s", ecrURIRoot, service, tag)
}

func ComposeNames(configs ServiceMap) []string {
	names := make([]string, 0)
	for name, s := range configs {
		var composeName string
		if s.ECR {
			composeName = name + "-ecr"
		} else {
			composeName = name
		}
		names = append(names, composeName)
	}

	return names
}

func RepoNames(services ServiceMap) []string {
	var repos []string

	for name, s := range services {
		if s.IsGithubRepo {
			repos = append(repos, name)
		}
	}

	return repos
}

func applyOverrides(services ServiceMap, overrides map[string]ServiceOverride) error {
	for name, override := range overrides {
		s, ok := services[name]
		if !ok {
			return fmt.Errorf("%s is not a valid service", name)
		}

		s.ECR = override.ECR
		if override.ECRTag != "" {
			s.ECRTag = override.ECRTag
		}

		services[name] = s
	}

	return nil
}
