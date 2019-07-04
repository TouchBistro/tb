package git

import (
	"fmt"
	"github.com/TouchBistro/tb/src/util"
)

func Clone(repoName string) error {
	repoURL := fmt.Sprintf("git@github.com:TouchBistro/%s.git", repoName)
	err := util.Exec("git", "clone", repoURL)
	return err
}

func Pull(repoName string) error {
	repoURL := fmt.Sprintf("git@github.com:TouchBistro/%s.git", repoName)
	err := util.Exec("git", "-C", repoURL, "pull")
	return err
}
