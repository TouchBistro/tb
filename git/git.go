package git

import (
	"fmt"

	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
)

func repoURL(repoName string) string {
	return fmt.Sprintf("git@github.com:TouchBistro/%s.git", repoName)
}

func Clone(repoName, destDir string) error {
	repoURL := repoURL(repoName)
	destPath := fmt.Sprintf("%s/%s", destDir, repoName)
	err := util.Exec("git", "clone", repoURL, destPath)

	return errors.Wrapf(err, "exec failed to clone %s to %s", repoName, destDir)
}

func Pull(repoName string) error {
	repoURL := repoURL(repoName)
	err := util.Exec("git", "-C", repoURL, "pull")

	return errors.Wrapf(err, "exec failed to pull %s", repoName)
}

func RepoNames(services map[string]config.Service) []string {
	var repos []string

	for name, s := range services {
		if s.IsGithubRepo {
			repos = append(repos, name)
		}
	}

	return repos
}
