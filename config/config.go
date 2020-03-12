package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/goutils/spinner"
	"github.com/TouchBistro/tb/app"
	"github.com/TouchBistro/tb/compose"
	"github.com/TouchBistro/tb/git"
	"github.com/TouchBistro/tb/login"
	"github.com/TouchBistro/tb/playlist"
	"github.com/TouchBistro/tb/registry"
	"github.com/TouchBistro/tb/service"
	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Package state for storing config info
var tbrc userConfig
var registryResult registry.RegistryResult

type InitOptions struct {
	LoadServices     bool
	LoadApps         bool
	UpdateRegistries bool
}

/* Paths */

func TBRootPath() string {
	return filepath.Join(os.Getenv("HOME"), ".tb")
}

func ReposPath() string {
	return filepath.Join(TBRootPath(), "repos")
}

func RegistriesPath() string {
	return filepath.Join(TBRootPath(), "registries")
}

func IOSBuildPath() string {
	return filepath.Join(TBRootPath(), "ios")
}

/* Config Accessors */

func LoginStategies() ([]login.LoginStrategy, error) {
	s, err := login.ParseStrategies(registryResult.LoginStrategies)
	return s, errors.Wrap(err, "Failed to parse login strategies")
}

func BaseImages() []string {
	return registryResult.BaseImages
}

func LoadedServices() *service.ServiceCollection {
	return registryResult.Services
}

func LoadedPlaylists() *playlist.PlaylistCollection {
	return registryResult.Playlists
}

func LoadedIOSApps() *app.AppCollection {
	return registryResult.IOSApps
}

func LoadedMacApps() *app.AppCollection {
	return registryResult.MacApps
}

/* Private Functions */

func setupEnv() error {
	tbRoot := TBRootPath()

	// Create $TB_ROOT directory if it doesn't exist
	if !file.FileOrDirExists(tbRoot) {
		err := os.Mkdir(tbRoot, 0755)
		if err != nil {
			return errors.Wrapf(err, "failed to create $TB_ROOT directory at %s", tbRoot)
		}
	}
	return nil
}

func cloneOrPullRegistry(r registry.Registry, shouldUpdate bool) error {
	if r.LocalPath != "" {
		return nil
	}

	// Clone if missing
	if !file.FileOrDirExists(r.Path) {
		log.Debugf("Registry %s is missing, cloning...", r.Name)
		err := git.Clone(r.Name, r.Path)
		if err != nil {
			return errors.Wrapf(err, "failed to clone registry to %s", r.Path)
		}

		return nil
	}

	if shouldUpdate {
		log.Debugf("Updating registry %s...", r.Name)
		err := git.Pull(r.Name, RegistriesPath())
		if err != nil {
			return errors.Wrapf(err, "failed to update registry %s", r.Name)
		}
	}

	return nil
}

/* Public Functions */

func Init(opts InitOptions) error {
	err := setupEnv()
	if err != nil {
		return errors.Wrap(err, "failed to setup tb environment")
	}

	if !IsExperimentalEnabled() {
		log.Debugln("Using legacy config init")
		return legacyInit()
	}

	// TODO scope if there's a way to pass lazydocker a custom tb specific config
	// Also consider creating a lazydocker package to abstract this logic so it doesn't seem so ad hoc
	// Create lazydocker config
	ldDirPath := filepath.Join(os.Getenv("HOME"), "Library/Application Support/jesseduffield/lazydocker")
	err = os.MkdirAll(ldDirPath, 0766)
	if err != nil {
		return errors.Wrapf(err, "failed to create lazydocker config directory %s", ldDirPath)
	}

	const lazydockerConfig = `
reporting: "off"
gui:
  wrapMainPanel: true
update:
  dockerRefreshInterval: 2000ms`

	ldConfigPath := filepath.Join(ldDirPath, "lazydocker.yml")
	err = ioutil.WriteFile(ldConfigPath, []byte(lazydockerConfig), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to create lazydocker config file")
	}

	// THE REGISTRY ZONE

	log.Infoln("Cloning/pulling registries...")
	if len(tbrc.Registries) == 0 {
		return errors.New("No registries defined in tbrc")
	}

	// Clone missing registries and pull existing ones
	successCh := make(chan string)
	failedCh := make(chan error)

	for _, r := range tbrc.Registries {
		go func(successCh chan string, failedCh chan error, r registry.Registry) {
			err := cloneOrPullRegistry(r, opts.UpdateRegistries)
			if err != nil {
				failedCh <- err
				return
			}

			successCh <- r.Name
		}(successCh, failedCh, r)
	}

	spinner.SpinnerWait(successCh, failedCh, "\r\t☑ finished cloning/pulling registry %s\n", "failed cloning/pulling registry", len(tbrc.Registries))

	registryResult, err = registry.ReadRegistries(tbrc.Registries, registry.ReadOptions{
		ShouldReadServices: opts.LoadServices,
		ShouldReadApps:     opts.LoadApps,
		RootPath:           TBRootPath(),
		ReposPath:          ReposPath(),
		Overrides:          tbrc.Overrides,
		CustomPlaylists:    tbrc.Playlists,
	})
	if err != nil {
		return errors.Wrap(err, "failed to read config files from registries")
	}

	if opts.LoadServices {
		// Create docker-compose.yml
		log.Debugln("Generating docker-compose.yml file...")

		composePath := filepath.Join(TBRootPath(), dockerComposePath)
		file, err := os.OpenFile(composePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return errors.Wrapf(err, "failed to open file %s", composePath)
		}
		defer file.Close()

		err = compose.CreateComposeFile(registryResult.Services, file)
		if err != nil {
			return errors.Wrap(err, "failed to generated docker-compose file")
		}
		log.Debugln("Successfully generated docker-compose.yml")
	}

	return nil
}

func CloneMissingRepos() error {
	log.Info("☐ checking ~/.tb directory for missing git repos for docker-compose.")

	repos := make([]string, 0)
	it := registryResult.Services.Iter()
	for it.HasNext() {
		s := it.Next()
		if s.HasGitRepo() {
			repos = append(repos, s.GitRepo)
		}
	}
	repos = util.UniqueStrings(repos)

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
