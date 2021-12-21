package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/tb/errkind"
)

const apiURL = "https://api.github.com"

// Client manages communication with the GitHub API.
type Client struct {
	httpClient *http.Client
}

// New creates a new Client using the given http client to make requests.
func New(httpClient *http.Client) *Client {
	return &Client{httpClient: httpClient}
}

// LatestReleaseTag retrieves the tag name for the latest release for the given repo.
func (c *Client) LatestReleaseTag(ctx context.Context, org, repo string) (string, error) {
	const op = errors.Op("github.Client.LatestReleaseTag")
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", apiURL, org, repo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", errors.Wrap(err, errors.Meta{
			Kind:   errkind.Internal,
			Reason: "failed to create GET request to GitHub API",
			Op:     op,
		})
	}
	// Use v3 API
	req.Header.Add("Accept", "application/vnd.github.v3+json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, errors.Meta{
			Kind:   errkind.GitHub,
			Reason: fmt.Sprintf("unable to get latest release of %s/%s", org, repo),
			Op:     op,
		})
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", errors.New(errkind.GitHub, fmt.Sprintf("got %d status from the GitHub API", resp.StatusCode), op)
	}

	var respBody struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return "", errors.Wrap(err, errors.Meta{
			Kind:   errkind.GitHub,
			Reason: "unable to parse response body",
			Op:     op,
		})
	}
	return respBody.TagName, nil
}
