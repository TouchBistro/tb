package git

import (
	"fmt"

	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/util"
)

func Clone(repoName string) error {
	repoURL := fmt.Sprintf("git@github.com:TouchBistro/%s.git", repoName)
	_, err := util.Exec("git", "clone", repoURL)
	return err
}

func Pull(repoName string) error {
	repoURL := fmt.Sprintf("git@github.com:TouchBistro/%s.git", repoName)
	_, err := util.Exec("git", "-C", repoURL, "pull")
	return err
}

func RepoNames(services *[]config.Service) []string {
	var repos []string

	for _, s := range *services {
		if s.IsGithubRepo {
			repos = append(repos, s.Name)
		}
	}

	return repos
}
