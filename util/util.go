package util

import (
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/TouchBistro/goutils/progress"
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

// OutputLogger implements progress.OutputLogger using logrus.
type OutputLogger struct {
	*logrus.Logger
}

func (ol OutputLogger) WithFields(fields progress.Fields) progress.Logger {
	return Logger{ol.Logger.WithFields(logrus.Fields(fields))}
}

func (ol OutputLogger) Output() io.Writer {
	return ol.Logger.Out
}

// Logger implements progress.Logger using logrus.
type Logger struct {
	logrus.FieldLogger
}

func (l Logger) WithFields(fields progress.Fields) progress.Logger {
	return Logger{l.FieldLogger.WithFields(logrus.Fields(fields))}
}

func (l Logger) Output() io.Writer {
	return l.FieldLogger.(*logrus.Logger).Out
}
