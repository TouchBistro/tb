package util

import (
	"os/exec"

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

	stdoutLogger := log.New()
	stdoutWriter := stdoutLogger.WriterLevel(log.DebugLevel)
	defer stdoutWriter.Close()

	stderrLogger := log.New()
	stderrWriter := stderrLogger.WriterLevel(log.WarnLevel)
	defer stderrWriter.Close()

	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter

	err := cmd.Run()
	if err != nil {
		log.Warnf("cmd.Run() failed with %s while running %s %s\n", err, name, arg)
		return err
	}

	return nil
}
