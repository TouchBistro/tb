package util

import (
	"crypto/md5"
	"os/exec"
	"strings"

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

	stdout := log.WithFields(log.Fields{
		"pipe":    "stdout",
		"command": name,
	}).WriterLevel(log.DebugLevel)
	defer stdout.Close()

	stderr := log.WithFields(log.Fields{
		"pipe":    "stderr",
		"command": name,
	}).WriterLevel(log.DebugLevel)
	defer stderr.Close()

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	if err != nil {
		return errors.Wrapf(err, "Exec failed to run %s %s", name, arg)
	}

	return nil
}

func StringToUpperAndSnake(str string) string {
	return strings.ReplaceAll(strings.ToUpper(str), "-", "_")
}

func MD5Checksum(buf []byte) ([]byte, error) {
	hash := md5.New()
	_, err := hash.Write(buf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to write to hash")
	}

	return hash.Sum(nil), nil
}
