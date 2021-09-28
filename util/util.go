package util

import (
	"fmt"
	"io"
	"regexp"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func IsMacOS() bool {
	return runtime.GOOS == "darwin"
}

func IsLinux() bool {
	return runtime.GOOS == "linux"
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

func ExpandVars(str string, vars map[string]string) (string, error) {
	// Regex to match variable substitution of the form ${VAR}
	regex := regexp.MustCompile(`\$\{([\w-@:]+)\}`)
	var result string

	lastEndIndex := 0
	for _, match := range regex.FindAllStringSubmatchIndex(str, -1) {
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
			var ok bool
			varValue, ok = vars[varName]
			if !ok {
				return "", fmt.Errorf("unknown variable %q", varName)
			}
		}

		result += str[lastEndIndex:startIndex]
		result += varValue
		lastEndIndex = endIndex
	}

	result += str[lastEndIndex:]
	return result, nil
}

func UniqueStrings(s []string) []string {
	set := make(map[string]bool)
	var us []string
	for _, v := range s {
		if ok := set[v]; ok {
			continue
		}

		set[v] = true
		us = append(us, v)
	}
	return us
}

// This is deprecated, use resource.ParseName.
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

// DiscardLogger returns a new logger that logs to nothing.
func DiscardLogger() logrus.FieldLogger {
	logger := logrus.New()
	logger.Out = io.Discard
	return logger
}
