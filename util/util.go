package util

import (
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
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

	stdOutLogger := log.New()
	stdOutWriter := stdOutLogger.WriterLevel(log.DebugLevel)
	defer stdOutWriter.Close()

	stdErrLogger := log.New()
	stdErrWriter := stdErrLogger.WriterLevel(log.WarnLevel)
	defer stdOutWriter.Close()

	cmd.Stdout = stdOutWriter
	cmd.Stderr = stdErrWriter

	err := cmd.Run()
	if err != nil {
		log.Warnf("cmd.Run() failed with %s while running %s %s\n", err, name, arg)
		return err
	}

	return nil
}

func FileOrDirExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func AppendLineToFile(path string, line string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(line + "\n")
	return err
}
