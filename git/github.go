package git

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	apiURL             = "https://api.github.com"
	githubTokenVarName = "GITHUB_TOKEN"
)

// CheckGithubAPIToken will check if the GITHUB_TOKEN env var is set.
// It also supports HOMEBREW_GITHUB_API_TOKEN but this is deprecated.
func CheckGithubAPIToken() {
	log.Debugf("Checking if %s is set", githubTokenVarName)
	token := os.Getenv(githubTokenVarName)
	if token != "" {
		log.Debugf("%s is set", token)
		return
	}

	// HOMEBREW_GITHUB_API_TOKEN is supported for backwards compatibility
	// but it's deprecated
	const varName = "HOMEBREW_GITHUB_API_TOKEN"
	token = os.Getenv(varName)
	if token == "" {
		return
	}

	log.Warnf("Using %s is deprecated. Please use %s instead.", varName, githubTokenVarName)
	os.Setenv(githubTokenVarName, token)
}

type getBranchResponse struct {
	Commit struct {
		Sha string `json:"sha"`
	} `json:"commit"`
}

func GetBranchHeadSha(repo, branch string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/branches/%s", apiURL, repo, branch)
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", errors.Wrap(err, "Failed to create GET request to GitHub API")
	}

	// Add auth token if it's set
	hasAuth := false
	if tokenVal := os.Getenv(githubTokenVarName); tokenVal != "" {
		token := fmt.Sprintf("token %s", tokenVal)
		req.Header.Add("Authorization", token)
		hasAuth = true
	}

	// Use v3 API
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	res, err := client.Do(req)
	if err != nil {
		return "", errors.Wrapf(err, "Unable to complete GET request %s", url)
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", errors.Wrap(err, "Unable to read response body")
	}

	if res.StatusCode != http.StatusOK {
		// GitHub responds with 404 instead of 401 if you don't have access to protect private repos
		if res.StatusCode == http.StatusNotFound {
			msg := fmt.Sprintf("GitHub repo: %s, branch: %s not found", repo, branch)

			// Try to help people out
			if !hasAuth {
				msg += fmt.Sprintf("\nIf it's a private repo you need to set %s to access it", githubTokenVarName)
			}

			return "", errors.New(msg)
		}

		return "", errors.Errorf("Got %d response from GitHub API:\n%s", res.StatusCode, string(body))
	}

	getBranchResp := getBranchResponse{}
	err = json.Unmarshal(body, &getBranchResp)
	if err != nil {
		return "", errors.Wrap(err, "Failed to parse JSON from reponse body")
	}

	sha := getBranchResp.Commit.Sha
	return sha, nil
}

func GetLatestRelease() (string, error) {
	url := fmt.Sprintf("%s/repos/TouchBistro/tb/releases/latest", apiURL)
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", errors.Wrap(err, "Failed to create GET request to GitHub API")
	}

	// Use v3 API
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	res, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "Unable to get latest release of tb")
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", errors.Wrap(err, "Unable to read response body")
	}

	if res.StatusCode != http.StatusOK {
		return "", errors.Errorf("Got %d response from GitHub API:\n%s", res.StatusCode, string(body))
	}

	var jsonDict map[string]interface{}
	err = json.Unmarshal(body, &jsonDict)
	if err != nil {
		return "", errors.Wrap(err, "Failed to parse JSON from reponse body")
	}

	tagName, ok := jsonDict["tag_name"].(string)
	if !ok {
		return "", errors.New("Unable to get tag name from response")
	}

	return tagName, nil
}
