package util

import (
	"runtime"
	"strings"
)

func IsMacOS() bool {
	return runtime.GOOS == "darwin"
}

func IsLinux() bool {
	return runtime.GOOS == "linux"
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

func DockerName(name string) string {
	// docker does not allow slashes in container names
	// so we'll replace them with dashes
	sanitized := strings.ReplaceAll(name, "/", "-")
	// docker does not allow upper case letters in image names
	// need to convert it all to lower case or docker-compose build breaks
	return strings.ToLower(sanitized)
}
