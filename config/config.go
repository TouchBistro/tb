package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/goutils/spinner"
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
var globalConfig registry.GlobalConfig
var services *service.ServiceCollection
var playlists *playlist.PlaylistCollection
var tbRoot string

type InitOptions struct {
	LoadServices     bool
	UpdateRegistries bool
}

/* Getters for private & computed vars */

func TBRootPath() string {
	return tbRoot
}

func ReposPath() string {
	return filepath.Join(tbRoot, "repos")
}

func RegistriesPath() string {
	return filepath.Join(tbRoot, "registries")
}

func LoginStategies() ([]login.LoginStrategy, error) {
	s, err := login.ParseStrategies(globalConfig.LoginStrategies)
	return s, errors.Wrap(err, "Failed to parse login strategies")
}

func BaseImages() []string {
	return globalConfig.BaseImages
}

func LoadedServices() *service.ServiceCollection {
	return services
}

func LoadedPlaylists() *playlist.PlaylistCollection {
	return playlists
}

/* Private functions */

func setupEnv() error {
	// Set $TB_ROOT so it works in the docker-compose file
	tbRoot = filepath.Join(os.Getenv("HOME"), ".tb")
	os.Setenv("TB_ROOT", tbRoot)

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
	isLocal := r.LocalPath != ""

	// Clone if missing and not local
	if !isLocal && !file.FileOrDirExists(r.Path) {
		log.Debugf("Registry %s is missing, cloning...", r.Name)
		err := git.Clone(r.Name, RegistriesPath())
		if err != nil {
			return errors.Wrapf(err, "failed to clone registry to %s", r.Path)
		}

		return nil
	}

	if !isLocal && shouldUpdate {
		log.Debugf("Updating registry %s...", r.Name)
		err := git.Pull(r.Name, RegistriesPath())
		if err != nil {
			return errors.Wrapf(err, "failed to update registry %s", r.Name)
		}
	}

	return nil
}

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

	log.Debugln("Resolving registries...")
	if len(tbrc.Registries) == 0 {
		return errors.New("No registries defined in tbrc")
	}

	// Clone missing registries and pull existing ones
	for _, r := range tbrc.Registries {
		err := cloneOrPullRegistry(r, opts.UpdateRegistries)
		if err != nil {
			return errors.Wrapf(err, "failed to resolve registry %s", r.Name)
		}
	}

	if opts.LoadServices {
		log.Debugln("Loading services...")

		serviceList := make([]service.Service, 0)
		playlistList := make([]playlist.Playlist, 0)

		for _, r := range tbrc.Registries {
			log.Debugf("Reading services from registry %s", r.Name)

			registryServices, conf, err := registry.ReadServices(r, TBRootPath(), ReposPath())
			if err != nil {
				return errors.Wrapf(err, "failed to read services for registry %s", r.Name)
			}

			log.Debugf("Reading playlists from registry %s", r.Name)
			registryPlaylists, err := registry.ReadPlaylists(r)
			if err != nil {
				return errors.Wrapf(err, "failed to read playlists for registry %s", r.Name)
			}

			globalConfig.BaseImages = append(globalConfig.BaseImages, conf.BaseImages...)
			globalConfig.LoginStrategies = append(globalConfig.LoginStrategies, conf.LoginStrategies...)

			serviceList = append(serviceList, registryServices...)
			playlistList = append(playlistList, registryPlaylists...)
		}

		// Dedup slices
		globalConfig.BaseImages = util.UniqueStrings(globalConfig.BaseImages)
		globalConfig.LoginStrategies = util.UniqueStrings(globalConfig.LoginStrategies)

		log.Debugln("Merging services and applying overrides...")
		services, err = service.NewServiceCollection(serviceList, tbrc.Overrides)
		if err != nil {
			return errors.Wrap(err, "failed to merge services into ServiceCollection")
		}

		log.Debugln("Merging playlists and custom playlists...")
		playlists, err = playlist.NewPlaylistCollection(playlistList, tbrc.Playlists)
		if err != nil {
			return errors.Wrap(err, "failed to merge playlists into PlaylistCollection")
		}

		// Create docker-compose.yml
		log.Debugln("Generating docker-compose.yml file...")

		composePath := filepath.Join(tbRoot, dockerComposePath)
		file, err := os.OpenFile(composePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return errors.Wrapf(err, "failed to open file %s", composePath)
		}
		defer file.Close()

		err = compose.CreateComposeFile(services, file)
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
	it := services.Iter()
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
