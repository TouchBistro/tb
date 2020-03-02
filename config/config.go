package config

import (
	"os"
	"path/filepath"

	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/goutils/spinner"
	"github.com/TouchBistro/tb/git"
	"github.com/TouchBistro/tb/login"
	"github.com/TouchBistro/tb/playlist"
	"github.com/TouchBistro/tb/service"
	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var tbrc userConfig
var serviceConfig ServiceConfig
var services *service.ServiceCollection
var playlists *playlist.PlaylistCollection
var tbRoot string

const (
	servicesPath             = "services.yml"
	playlistPath             = "playlists.yml"
	dockerComposePath        = "docker-compose.yml"
	localstackEntrypointPath = "localstack-entrypoint.sh"
	lazydockerConfigPath     = "lazydocker.yml"
)

/* Getters for private & computed vars */

func TBRootPath() string {
	return tbRoot
}

func ReposPath() string {
	return filepath.Join(tbRoot, "repos")
}

func LoginStategies() ([]login.LoginStrategy, error) {
	s, err := login.ParseStrategies(serviceConfig.Global.LoginStategies)
	return s, errors.Wrap(err, "Failed to parse login strategies")
}

func BaseImages() []string {
	return serviceConfig.Global.BaseImages
}

func Services() *service.ServiceCollection {
	return services
}

func Playlists() *playlist.PlaylistCollection {
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

func Init() error {
	err := setupEnv()
	if err != nil {
		return errors.Wrap(err, "failed to setup $TB_ROOT env")
	}

	return legacyInit()
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
