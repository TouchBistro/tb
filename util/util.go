package util

import (
	"crypto/md5"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

func MD5Checksum(buf []byte) ([]byte, error) {
	hash := md5.New()
	_, err := hash.Write(buf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to write to hash")
	}

	return hash.Sum(nil), nil
}

func Prompt(msg string) bool {
	// check for yes and assume no on any other input to avoid annoyance
	fmt.Print(msg)
	var resp string
	_, err := fmt.Scanln(&resp)
	if err != nil {
		return false
	}
	if strings.ToLower(string(resp[0])) == "y" {
		return true
	}
	return false
}

func ExpandVars(str string, vars map[string]string) string {
	// Regex to match variable substitution of the form ${VAR}
	regex := regexp.MustCompile(`\$\{([\w-@:]+)\}`)
	indices := regex.FindAllStringSubmatchIndex(str, -1)

	// Go through the string in reverse order and replace all variables with their value
	expandedStr := str
	for i := len(indices) - 1; i >= 0; i-- {
		match := indices[i]
		// match[0] is the start index of the whole match
		startIndex := match[0]
		// match[1] is the end index of the whole match (exclusive)
		endIndex := match[1]
		// match[2] is start index of group
		startIndexGroup := match[2]
		// match[3] is end index of group (exclusive)
		endIndexGroup := match[3]

		varName := str[startIndexGroup:endIndexGroup]
		const envPrefix = "@env:"
		var varValue string
		if strings.HasPrefix(varName, envPrefix) {
			varValue = fmt.Sprintf("${%s}", strings.TrimPrefix(varName, envPrefix))
		} else {
			varValue = vars[varName]
		}

		expandedStr = expandedStr[:startIndex] + varValue + expandedStr[endIndex:]
	}

	return expandedStr
}

func UniqueStrings(s []string) []string {
	set := make(map[string]bool)
	us := make([]string, 0)

	for _, v := range s {
		if _, ok := set[v]; ok {
			continue
		}

		set[v] = true
		us = append(us, v)
	}

	return us
}

func SplitNameParts(name string) (string, string, error) {
	// Full form of item name in a registry is
	// <org>/<repo>/<item> where an item is a service, playlist or app
	regex := regexp.MustCompile(`^(?:([\w-]+\/[\w-]+)\/)?([\w-]+)$`)
	matches := regex.FindStringSubmatch(name)

	if len(matches) == 0 {
		return "", "", errors.Errorf("%s is not a valid tb item name. Format is <org>/<repo>/<item>", name)
	}

	return matches[1], matches[2], nil
}

func DockerName(name string) string {
	// docker does not allow slashes in container names
	// so we'll replace them with dashes
	sanitized := strings.ReplaceAll(name, "/", "-")
	// docker does not allow upper case letters in image names
	// need to convert it all to lower case or docker-compose build breaks
	return strings.ToLower(sanitized)
}
