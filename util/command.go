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

func ExecStdoutStderr(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	return err
}
