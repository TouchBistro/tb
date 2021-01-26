package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/TouchBistro/goutils/file"
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

func DesktopAppsPath() string {
	return filepath.Join(TBRootPath(), "desktop")
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

func LoadedDesktopApps() *app.AppCollection {
	return registryResult.DesktopApps
}

/* Private Functions */

func cloneOrPullRegistry(r registry.Registry, shouldUpdate bool) error {
	if r.LocalPath != "" {
		return nil
	}

	// Clone if missing
	if !file.Exists(r.Path) {
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
	tbRoot := TBRootPath()

	// Create ~/.tb directory if it doesn't exist
	if err := os.MkdirAll(tbRoot, 0o755); err != nil {
		return errors.Wrapf(err, "failed to create tb root directory at %s", tbRoot)
	}

	// TODO scope if there's a way to pass lazydocker a custom tb specific config
	// Also consider creating a lazydocker package to abstract this logic so it doesn't seem so ad hoc
	// Create lazydocker config
	var ldDirPath string
	if util.IsMacOS() {
		ldDirPath = filepath.Join(os.Getenv("HOME"), "Library/Application Support/jesseduffield/lazydocker")
	} else {
		ldDirPath = filepath.Join(os.Getenv("HOME"), ".config/jesseduffield/lazydocker")
	}

	err := os.MkdirAll(ldDirPath, 0766)
	if err != nil {
		return errors.Wrapf(err, "failed to create lazydocker config directory %s", ldDirPath)
	}

	const lazydockerConfig = `
reporting: "off"
gui:
  wrapMainPanel: true
update:
  dockerRefreshInterval: 2000ms`

	ldConfigPath := filepath.Join(ldDirPath, "config.yml")
	err = ioutil.WriteFile(ldConfigPath, []byte(lazydockerConfig), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to create lazydocker config file")
	}

	git.CheckGithubAPIToken()

	// THE REGISTRY ZONE

	if len(tbrc.Registries) == 0 {
		return errors.New("No registries defined in tbrc")
	}

	// Clone missing registries and pull existing ones
	successCh := make(chan string)
	failedCh := make(chan error)
	// TODO(@cszatmary): For when we switch to the new spinner
	// s := spinner.New(
	// 	spinner.WithStartMessage("Cloning/pulling registries"),
	// 	spinner.WithStopMessage("☑ Finished cloning/pulling registries"),
	// 	spinner.WithCount(len(tbrc.Registries)),
	// 	spinner.WithPersistMessages(log.IsLevelEnabled(log.DebugLevel)),
	// )
	// logger := log.New()
	// logger.SetFormatter(log.StandardLogger().Formatter)
	// logger.SetOutput(s)
	// logger.SetLevel(log.StandardLogger().GetLevel())

	for _, r := range tbrc.Registries {
		go func(r registry.Registry) {
			err := cloneOrPullRegistry(r, opts.UpdateRegistries)
			if err != nil {
				failedCh <- err
				return
			}
			successCh <- r.Name
		}(r)
	}

	// TODO(@cszatmary): For when we switch to the new spinner
	// s.Start()
	// for i := 0; i < len(tbrc.Registries); i++ {
	// 	select {
	// 	case name := <-successCh:
	// 		s.IncWithMessagef("☑ finished cloning/pulling registry %s", name)
	// 	case err := <-failedCh:
	// 		return errors.Wrap(err, "failed cloning/pulling registry")
	// 	case <-time.After(time.Minute * 5):
	// 		return errors.New("timed out while cloning/pulling registries")
	// 	}
	// }
	// s.Stop()
	util.SpinnerWait(successCh, failedCh, "\r\t☑ finished cloning/pulling registry %s\n", "failed cloning/pulling registry", len(tbrc.Registries))

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

		composePath := filepath.Join(TBRootPath(), "docker-compose.yml")
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

func CloneOrPullRepos(shouldPull bool) error {
	log.Debug("checking ~/.tb directory for missing git repos for docker-compose.")
	repoSet := make(map[string]bool)
	it := registryResult.Services.Iter()
	for it.HasNext() {
		s := it.Next()
		if s.HasGitRepo() {
			repoSet[s.GitRepo.Name] = true
		}
	}

	// Figure out what actions (if any) are required for each repo
	// We need to make sure all repos are cloned in order to resolve of all the
	// references in the compose files to files in the repos.
	type action struct {
		repo  string
		clone bool
	}
	var actions []action
	for repo := range repoSet {
		log.Debugf("Checking repo %s", repo)
		repoPath := filepath.Join(ReposPath(), repo)
		if !file.Exists(repoPath) {
			actions = append(actions, action{repo, true})
			continue
		}

		// Hack to make sure repo was cloned properly
		// Sometimes it doesn't clone properly if the user does control-c during cloning
		// Figure out a better way to do this
		dirlen, err := file.DirLen(repoPath)
		if err != nil {
			return errors.Wrapf(err, "could not read project directory for %s", repo)
		}
		if dirlen <= 2 {
			// Directory exists but only contains .git subdirectory, rm and clone again below
			if err := os.RemoveAll(repoPath); err != nil {
				return errors.Wrapf(err, "could not remove project directory for %s", repo)
			}
			actions = append(actions, action{repo, true})
			continue
		}
		if shouldPull {
			actions = append(actions, action{repo, false})
		}
	}

	// successCh := make(chan action)
	successCh := make(chan string)
	failedCh := make(chan error)
	// TODO(@cszatmary): For when we switch to the new spinner
	// s := spinner.New(
	// 	spinner.WithStartMessage("Cloning/pulling service git repositories"),
	// 	spinner.WithStopMessage("☑ Finished cloning/pulling service git repositories"),
	// 	spinner.WithCount(len(actions)),
	// 	spinner.WithPersistMessages(log.IsLevelEnabled(log.DebugLevel)),
	// )
	// logger := log.New()
	// logger.SetFormatter(log.StandardLogger().Formatter)
	// logger.SetOutput(s)
	// logger.SetLevel(log.StandardLogger().GetLevel())

	for _, a := range actions {
		go func(a action) {
			var err error
			if a.clone {
				log.Debugf("\t☐ %s is missing. cloning git repo\n", a.repo)
				err = git.Clone(a.repo, filepath.Join(ReposPath(), a.repo))
			} else {
				log.Debugf("\t☐ %s exists. pulling git repo\n", a.repo)
				err = git.Pull(a.repo, ReposPath())
			}
			if err != nil {
				failedCh <- err
				return
			}
			successCh <- a.repo
		}(a)
	}

	// TODO(@cszatmary): For when we switch to the new spinner
	// s.Start()
	// for i := 0; i < len(tbrc.Registries); i++ {
	// 	select {
	// 	case a := <-successCh:
	// 		if a.clone {
	// 			s.IncWithMessagef("☑ finished cloning git repository %s", a.repo)
	// 		} else {
	// 			s.IncWithMessagef("☑ finished pulling git repository %s", a.repo)
	// 		}
	// 	case err := <-failedCh:
	// 		return errors.Wrap(err, "failed cloning/pulling git repository")
	// 	case <-time.After(time.Minute * 10):
	// 		return errors.New("timed out while cloning/pulling git repositories")
	// 	}
	// }
	// s.Stop()
	util.SpinnerWait(successCh, failedCh, "\r\t☑ finished cloning/pulling %s\n", "failed cloning/pulling git repo", len(actions))

	log.Info("☑ finished checking git repos")
	return nil
}
