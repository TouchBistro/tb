package util

import (
	"os/exec"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func IsCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error(), "command": command}).Debug("Error looking up command.")
		return false
	}
	return true
}

func Exec(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)

	// TODO: Tag each logger with the command its trying to exec and whether its the stderr or stdout logger
	stdoutLogger := log.New()
	stdoutWriter := stdoutLogger.WriterLevel(log.DebugLevel)
	defer stdoutWriter.Close()

	stderrLogger := log.New()
	stderrWriter := stderrLogger.WriterLevel(log.DebugLevel)
	defer stderrWriter.Close()

	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter

	err := cmd.Run()
	if err != nil {
		return errors.Wrapf(err, "Exec failed to run %s %s", name, arg)
	}

	return nil
}
