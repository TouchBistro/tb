package git

import (
	"context"

	"github.com/google/go-github/v37/github"
	"github.com/pkg/errors"
)

func GetLatestRelease() (string, error) {
	client := github.NewClient(nil)

	release, _, err := client.Repositories.GetLatestRelease(context.Background(), "TouchBistro", "tb")
	if err != nil {
		return "", errors.Wrap(err, "Failed to retrieve latest release from GitHub")
	}

	return *release.TagName, nil
}
