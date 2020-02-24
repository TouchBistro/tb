package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/goutils/spinner"
	"github.com/TouchBistro/tb/git"
	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

/* Types */

type volume struct {
	Value   string `yaml:"value"`
	IsNamed bool   `yaml:"named"`
}

type build struct {
	Args           map[string]string `yaml:"args"`
	Command        string            `yaml:"command"`
	DockerfilePath string            `yaml:"dockerfilePath"`
	Target         string            `yaml:"target"`
	Volumes        []volume          `yaml:"volumes"`
}

type remote struct {
	Command string   `yaml:"command"`
	Enabled bool     `yaml:"enabled"`
	Image   string   `yaml:"image"`
	Tag     string   `yaml:"tag"`
	Volumes []volume `yaml:"volumes"`
}

type Service struct {
	Build        build             `yaml:"build"`
	Dependencies []string          `yaml:"dependencies"`
	Entrypoint   []string          `yaml:"entrypoint"`
	EnvFile      string            `yaml:"envFile"`
	EnvVars      map[string]string `yaml:"envVars"`
	GitRepo      string            `yaml:"repo"`
	Ports        []string          `yaml:"ports"`
	PreRun       string            `yaml:"preRun"`
	Remote       remote            `yaml:"remote"`
	// Extra properties added at runtime
	RecipeName string `yaml:"-"`
}

type ServiceMap map[string]Service

type ServiceListMap map[string][]Service

type RecipeServiceConfig struct {
	Global struct {
		BaseImages      []string          `yaml:"baseImages"`
		LoginStrategies []string          `yaml:"loginStrategies"`
		Variables       map[string]string `yaml:"variables"`
	} `yaml:"global"`
	Services ServiceMap `yaml:"services"`
}

type ServiceConfig struct {
	BaseImages      []string
	LoginStrategies []string
	Services        ServiceListMap
}

/* Methods & computed properties */

func (s Service) HasGitRepo() bool {
	return s.GitRepo != ""
}

func (s Service) UseRemote() bool {
	return s.Remote.Enabled
}

func (s Service) CanBuild() bool {
	return s.Build.DockerfilePath != ""
}

func (s Service) ImageURI() string {
	if s.Remote.Tag == "" {
		return s.Remote.Image
	}

	return fmt.Sprintf("%s:%s", s.Remote.Image, s.Remote.Tag)
}

func (sm ServiceMap) Names() []string {
	names := make([]string, 0, len(sm))
	for name := range sm {
		names = append(names, name)
	}

	return names
}

func (slm ServiceListMap) Get(name string) (string, Service, error) {
	recipeName, serviceName, err := recipeNameParts(name)
	if err != nil {
		return "", Service{}, errors.Wrapf(err, "invalid service name %s", name)
	}

	list, ok := slm[serviceName]
	if !ok {
		return "", Service{}, errors.Errorf("No such service %s", serviceName)
	}

	if recipeName == "" {
		if len(list) > 1 {
			return "", Service{}, errors.Errorf("Multiple services named %s found. Please specify the recipe the service belongs to.", serviceName)
		}

		s := list[0]
		return joinNameParts(s.RecipeName, serviceName), s, nil
	}

	for _, service := range list {
		if service.RecipeName == recipeName {
			return name, service, nil
		}
	}

	return "", Service{}, errors.Errorf("No such service %s", name)
}

func (slm ServiceListMap) Set(name string, value Service) error {
	recipeName, serviceName, err := recipeNameParts(name)
	if err != nil {
		return errors.Wrapf(err, "invalid service name %s", name)
	}

	list, ok := slm[serviceName]
	if !ok {
		return errors.Errorf("No such service %s", serviceName)
	}

	if recipeName == "" {
		if len(list) > 1 {
			return errors.Errorf("Multiple services named %s found. Please specify the recipe the service belongs to.", serviceName)
		}

		slm[serviceName][0] = value
		return nil
	}

	for i, service := range list {
		if service.RecipeName == recipeName {
			slm[serviceName][i] = value
			return nil
		}
	}

	return errors.Errorf("No such service %s", name)
}

func (slm ServiceListMap) ServiceMap() ServiceMap {
	sm := make(ServiceMap)
	for n, list := range slm {
		for _, s := range list {
			sm[joinNameParts(s.RecipeName, n)] = s
		}
	}

	return sm
}

/* Private helpers */

func parseServices(config RecipeServiceConfig) (ServiceMap, error) {
	parsedServices := make(ServiceMap)

	// Validate each service and perform any necessary actions
	for name, service := range config.Services {
		// Make sure either local or remote usage is specified
		if !service.CanBuild() && service.Remote.Image == "" {
			msg := fmt.Sprintf("Must specify at least one of 'build.dockerfilePath' or 'remote.image' for service %s", name)
			return nil, errors.New(msg)
		}

		// Make sure repo is specified if not using remote
		if !service.UseRemote() && !service.CanBuild() {
			msg := fmt.Sprintf("'remote.enabled: false' is set but 'build.dockerfilePath' was not provided for service %s", name)
			return nil, errors.New(msg)
		}

		// Set special service specific vars
		vars := config.Global.Variables
		vars["@ROOTPATH"] = TBRootPath()

		if service.HasGitRepo() {
			vars["@REPOPATH"] = filepath.Join(ReposPath(), service.GitRepo)
		}

		// Expand any vars
		service.Build.DockerfilePath = util.ExpandVars(service.Build.DockerfilePath, vars)
		service.EnvFile = util.ExpandVars(service.EnvFile, vars)
		service.Remote.Image = util.ExpandVars(service.Remote.Image, vars)

		for key, value := range service.EnvVars {
			service.EnvVars[key] = util.ExpandVars(value, vars)
		}

		for i, volume := range service.Build.Volumes {
			service.Build.Volumes[i].Value = util.ExpandVars(volume.Value, vars)
		}

		for i, volume := range service.Remote.Volumes {
			service.Remote.Volumes[i].Value = util.ExpandVars(volume.Value, vars)
		}

		parsedServices[name] = service
	}

	return parsedServices, nil
}

func applyOverrides(services ServiceListMap, overrides map[string]serviceOverride) (ServiceListMap, error) {
	newServices := make(ServiceListMap)
	for name, list := range services {
		newList := make([]Service, len(list))
		copy(newList, list)
		newServices[name] = newList
	}

	for name, override := range overrides {
		_, s, err := services.Get(name)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get service matching override %s", name)
		}

		// Validate overrides
		if override.Remote.Enabled && s.Remote.Image == "" {
			msg := fmt.Sprintf("remote.enabled is overridden to true for %s but it is not available from a remote source", name)
			return nil, errors.New(msg)
		} else if !override.Remote.Enabled && !s.HasGitRepo() {
			msg := fmt.Sprintf("remote.enabled is overridden to false but %s cannot be built locally", name)
			return nil, errors.New(msg)
		}

		// Apply overrides to service
		if override.Build.Command != "" {
			s.Build.Command = override.Build.Command
		}

		if override.Build.Target != "" {
			s.Build.Target = override.Build.Target
		}

		if override.EnvVars != nil {
			for v, val := range override.EnvVars {
				s.EnvVars[v] = val
			}
		}

		if override.PreRun != "" {
			s.PreRun = override.PreRun
		}

		if override.Remote.Command != "" {
			s.Remote.Command = override.Remote.Command
		}

		s.Remote.Enabled = override.Remote.Enabled
		if override.Remote.Tag != "" {
			s.Remote.Tag = override.Remote.Tag
		}

		err = newServices.Set(name, s)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to set service named %s", name)
		}
	}

	return newServices, nil
}

/* Public funtions */

func CloneMissingRepos(services ServiceMap) error {
	log.Info("☐ checking ~/.tb directory for missing git repos for docker-compose.")

	repos := Repos(services)

	successCh := make(chan string)
	failedCh := make(chan error)

	count := 0
	// We need to clone every repo to resolve of all the references in the compose files to files in the repos.
	for _, repo := range repos {
		path := filepath.Join(ReposPath(), repo)

		if file.FileOrDirExists(path) {
			dirlen, err := file.DirLen(path)
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

	spinner.SpinnerWait(successCh, failedCh, "\r\t☑ finished cloning %s\n", "failed cloning git repo", count)

	log.Info("☑ finished checking git repos")
	return nil
}

func Repos(services ServiceMap) []string {
	var repos []string
	seenRepos := make(map[string]bool)

	for _, s := range services {
		if !s.HasGitRepo() {
			continue
		}
		repo := s.GitRepo

		// repo has already been added to the list, don't add it again
		if seenRepos[repo] {
			continue
		}

		repos = append(repos, repo)
		seenRepos[repo] = true
	}

	return repos
}
