package git

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/pkg/errors"
)

const (
	apiURL   = "https://api.github.com"
	tokenVar = "HOMEBREW_GITHUB_API_TOKEN"
)

func GetLatestRelease() (string, error) {
	url := fmt.Sprintf("%s/repos/TouchBistro/tb/releases/latest", apiURL)
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", errors.Wrap(err, "Failed to create GET request to GitHub API")
	}

	token := fmt.Sprintf("token %s", os.Getenv(tokenVar))
	req.Header.Add("Authorization", token)
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

	if res.StatusCode != 200 {
		return "", errors.New(fmt.Sprintf("Got %d response from GitHub API:\n%s", res.StatusCode, string(body)))
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
