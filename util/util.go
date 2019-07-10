package util

import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"sync"
)

func IsCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error(), "command": command}).Debug("Error looking up command.")
		return false
	}
	return true
}

func Exec(name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)
	cmd.Stdin = os.Stdin

	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	var errStdout, errStderr error
	stdout := io.MultiWriter(os.Stdout, &stdoutBuf)
	stderr := io.MultiWriter(os.Stderr, &stderrBuf)
	err := cmd.Start()
	if err != nil {
		log.Debugf("cmd.Start() failed with '%s'\n", err)
		return "", err
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		_, errStdout = io.Copy(stdout, stdoutIn)
		wg.Done()
	}()

	_, errStderr = io.Copy(stderr, stderrIn)
	wg.Wait()

	err = cmd.Wait()
	if err != nil {
		log.Warnf("cmd.Run() failed with %s while running %s %s\n", err, name, arg)
		return "", err
	}

	if errStdout != nil || errStderr != nil {
		return "", fmt.Errorf("failed to capture stdout or stderr while running %s %s", name, arg)
	}

	stdOut := stdoutBuf.String()
	stdErr := stderrBuf.String()

	// TODO: Not sure if this is a good idea - do we want to return an error if the stdErr buffer received data or just print it?
	if len(stdErr) != 0 {
		log.Warnf("cmd.Run() wrote to stdErr while running %s. Error: %s\n", name, stdErr)
	}

	return stdOut, nil
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
