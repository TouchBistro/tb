package config

import (
	"fmt"
	"os"
	"regexp"

	"github.com/pkg/errors"

	"github.com/TouchBistro/tb/git"
	"github.com/TouchBistro/tb/util"
	log "github.com/sirupsen/logrus"
)

/* Types */

type Service struct {
	GithubRepo string `yaml:"repo"`
	Migrations bool   `yaml:"migrations"`
	Remote     struct {
		Enabled bool   `yaml:"enabled"`
		Image   string `yaml:"image"`
		Tag     string `yaml:"tag"`
	} `yaml:"remote"`
}

type ServiceOverride struct {
	ECR    bool   `yaml:"ecr"`
	ECRTag string `yaml:"ecrTag"`
}

type ServiceMap map[string]Service

type ServiceConfig struct {
	Global struct {
		Variables map[string]string `yaml:"variables"`
	} `yaml:"global"`
	Services ServiceMap `yaml:"services"`
}

/* Methods & computed properties */

func (s Service) IsGithubRepo() bool {
	return s.GithubRepo != ""
}

func (s Service) UseRemote() bool {
	return s.Remote.Enabled
}

func (s Service) ImageURI() string {
	if s.Remote.Tag == "" {
		return s.Remote.Image
	}

	return fmt.Sprintf("%s:%s", s.Remote.Image, s.Remote.Tag)
}

/* Private helpers */

func expandVars(str string, vars map[string]string) string {
	// Regex to match variable substitution
	regex := regexp.MustCompile(`\$\{([\w-]+)\}`)
	indices := regex.FindAllStringSubmatchIndex(str, -1)

	// Go through the string in reverse order and replace all variables with their value
	expandedStr := str
	for i := len(indices) - 1; i >= 0; i-- {
		match := indices[i]
		// match[0] is the start index of the whole match
		startIndex := match[0]
		// match[1] is the end index of the whole match (exclusive)
		endIndex := match[1]
		// match[2] is start index of group
		startIndexGroup := match[2]
		// match[3] is end index of group (exclusive)
		endIndexGroup := match[3]

		varName := str[startIndexGroup:endIndexGroup]
		expandedStr = expandedStr[:startIndex] + vars[varName] + expandedStr[endIndex:]
	}

	return expandedStr
}

func parseServices(config ServiceConfig) (ServiceMap, error) {
	parsedServices := make(ServiceMap)

	// Validate each service and perform any necessary actions
	for name, service := range config.Services {
		// Make sure either local or remote usage is specified
		if !service.IsGithubRepo() && service.Remote.Image == "" {
			msg := fmt.Sprintf("Must specify at least one of 'repo' or 'remote.image' for service %s", name)
			return nil, errors.New(msg)
		}

		// Make sure repo is specified if not using remote
		if !service.UseRemote() && !service.IsGithubRepo() {
			msg := fmt.Sprintf("'enabled: false' is set but 'repo' was not provided for service %s", name)
			return nil, errors.New(msg)
		}

		// Expand any docker registry vars
		service.Remote.Image = expandVars(service.Remote.Image, config.Global.Variables)

		parsedServices[name] = service
	}

	return parsedServices, nil
}

func applyOverrides(services ServiceMap, overrides map[string]ServiceOverride) (ServiceMap, error) {
	newServices := make(ServiceMap)
	for name, s := range services {
		newServices[name] = s
	}

	for name, override := range overrides {
		s, ok := services[name]
		if !ok {
			return nil, fmt.Errorf("%s is not a valid service", name)
		}

		// Validate overrides
		if override.ECR && !s.UseRemote() {
			msg := fmt.Sprintf("ecr is overridden to true for %s but it is not available from a remote source", name)
			return nil, errors.New(msg)
		} else if !override.ECR && !s.IsGithubRepo() {
			msg := fmt.Sprintf("ecr is overridden to false but %s cannot be built locally", name)
			return nil, errors.New(msg)
		}

		// Apply overrides to service
		s.Remote.Enabled = override.ECR
		if override.ECRTag != "" {
			s.Remote.Tag = override.ECRTag
		}

		newServices[name] = s
	}

	return newServices, nil
}

/* Public funtions */

func ComposeName(name string, s Service) string {
	if s.UseRemote() && s.IsGithubRepo() {
		return name + "-remote"
	}

	return name
}

func CloneMissingRepos(services ServiceMap) error {
	log.Info("☐ checking ~/.tb directory for missing git repos for docker-compose.")

	repos := Repos(services)

	successCh := make(chan string)
	failedCh := make(chan error)

	count := 0
	// We need to clone every repo to resolve of all the references in the compose files to files in the repos.
	for _, repo := range repos {
		path := fmt.Sprintf("%s/%s", ReposPath(), repo)

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
		go func(successCh chan string, failedCh chan error, repo, destPath string) {
			err := git.Clone(repo, destPath)
			if err != nil {
				failedCh <- err
			} else {
				successCh <- repo
			}
		}(successCh, failedCh, repo, path)
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

func Repos(services ServiceMap) []string {
	var repos []string
	seenRepos := make(map[string]bool)

	for _, s := range services {
		if !s.IsGithubRepo() {
			continue
		}
		repo := s.GithubRepo

		// repo has already been added to the list, don't add it again
		if seenRepos[repo] {
			continue
		}

		repos = append(repos, repo)
		seenRepos[repo] = true
	}

	return repos
}
