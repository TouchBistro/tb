package util

import (
	"os"
	"os/exec"
)

func IsCommandAvailable(command string) bool {
	cmd := exec.Command("/bin/sh", "-c", "command -v "+command)
	err := cmd.Run()
	if err != nil {
		return false
	}
	return true
}

func Exec(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)

	// TODO: Pass an io.Writer / io.Reader for each, use os as defaults
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	return err
}
