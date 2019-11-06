package config

import (
	"fmt"
	"github.com/pkg/errors"
	"os"

	"github.com/TouchBistro/tb/git"
	"github.com/TouchBistro/tb/util"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	GithubRepo     string `yaml:"repo"`
	Migrations     bool   `yaml:"migrations"`
	ECR            bool   `yaml:"ecr"`
	ECRTag         string `yaml:"ecrTag"`
	DockerhubImage string `yaml:"dockerhubImage"`
}

type ServiceOverride struct {
	ECR    bool   `yaml:"ecr"`
	ECRTag string `yaml:"ecrTag"`
}

type ServiceMap = map[string]Service

func (s Service) IsGithubRepo() bool {
	return s.GithubRepo != ""
}

func ComposeName(name string, s Service) string {
	if s.ECR {
		return name + "-ecr"
	}

	return name
}

func ResolveEcrURI(service, tag string) string {
	return fmt.Sprintf("%s/%s:%s", ecrURIRoot, service, tag)
}

func CloneMissingRepos(services ServiceMap) error {
	log.Info("☐ checking ~/.tb directory for missing git repos for docker-compose.")

	repos := RepoNames(services)

	successCh := make(chan string)
	failedCh := make(chan error)

	count := 0
	// We need to clone every repo to resolve of all the references in the compose files to files in the repos.
	for _, repo := range repos {
		path := fmt.Sprintf("%s/%s", TBRootPath(), repo)

		if util.FileOrDirExists(path) {
			dirlen, err := util.DirLen(path)
			if err != nil {
				return errors.Wrap(err, "Could not read project directory")
			}
			// Directory exists but only contains .git subdirectory, rm and clone again
			if dirlen > 2 {
				continue
			}
			err = os.RemoveAll(path)
			if err != nil {
				return errors.Wrapf(err, "Couldn't remove project directory for %s", path)
			}
		}

		log.Debugf("\t☐ %s is missing. cloning git repo\n", repo)
		go func(successCh chan string, failedCh chan error, repo, root string) {
			err := git.Clone(repo, root)
			if err != nil {
				failedCh <- err
			} else {
				successCh <- repo
			}
		}(successCh, failedCh, repo, TBRootPath())
		count++
	}

	util.SpinnerWait(successCh, failedCh, "\r\t☑ finished cloning %s\n", "failed cloning git repo", count)

	log.Info("☑ finished checking git repos")
	return nil
}

func ComposeNames(configs ServiceMap) []string {
	names := make([]string, 0)
	for name, s := range configs {
		names = append(names, ComposeName(name, s))
	}

	return names
}

func RepoNames(services ServiceMap) []string {
	var repos []string
	repoNames := make(map[string]bool)

	for _, s := range services {
		repoName := s.GithubRepo
		if !s.IsGithubRepo() {
			continue
		}

		// repoName has already been added to the list, don't add it again
		if repoNames[repoName] {
			continue
		}

		repos = append(repos, repoName)
		repoNames[repoName] = true
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
