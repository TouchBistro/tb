package util

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/TouchBistro/tb/fatal"
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

func Exec(id string, name string, arg ...string) error {
	cmd := exec.Command(name, arg...)

	stdout := log.WithFields(log.Fields{
		"id": id,
	}).WriterLevel(log.DebugLevel)
	defer stdout.Close()

	stderr := log.WithFields(log.Fields{
		"id": id,
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

func ExecResult(id string, name string, args ...string) (*bytes.Buffer, error) {
	cmd := exec.Command(name, args...)

	stdoutBuf := &bytes.Buffer{}
	stderr := log.WithFields(log.Fields{
		"id": id,
	}).WriterLevel(log.DebugLevel)
	defer stderr.Close()

	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderr

	err := cmd.Run()
	if err != nil {
		return nil, errors.Wrapf(err, "Exec failed to run %s %s", name, args)
	}

	return stdoutBuf, nil
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

func spinnerBar(total int) func(int) {
	spinnerFrames := []string{"|", "/", "-", "\\"}
	progress := 0
	animState := 0
	return func(inc int) {
		progress += inc
		var bar strings.Builder
		bar.WriteString("\r")
		bar.WriteString(spinnerFrames[animState])
		bar.WriteString(" [")
		for i := 0; i < total; i++ {
			if progress > i {
				bar.WriteString("#")
			} else {
				bar.WriteString("-")
			}
		}
		bar.WriteString("]")
		animState++
		animState = animState % len(spinnerFrames)
		fmt.Print(bar.String())
		if progress == total {
			clearLine(total + 4)
		}
	}
}

func SpinnerWait(successCh chan string, failedCh chan error, successMsg string, failedMsg string, count int) {
	spin := spinnerBar(count)
	for i := 0; i < count; {
		select {
		case name := <-successCh:
			if !log.IsLevelEnabled(log.DebugLevel) {
				clearLine(count + 4)
			}
			log.Infof(successMsg, name)
			i++
			if !log.IsLevelEnabled(log.DebugLevel) {
				spin(1)
			}
		case err := <-failedCh:
			fmt.Printf("\r\n")
			fatal.ExitErrf(err, failedMsg)
		case <-time.After(time.Second / 10):
			if !log.IsLevelEnabled(log.DebugLevel) {
				spin(0)
			}
		}
	}
}

func clearLine(length int) {
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteString(" ")
	}
	fmt.Printf("\r")
	fmt.Print(b.String())
	fmt.Printf("\r")
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

func ExpandVars(str string, vars map[string]string) string {
	// Regex to match variable substitution of the form ${VAR}
	regex := regexp.MustCompile(`\$\{([\w-]+)\}`)
	indices := regex.FindAllStringSubmatchIndex(str, -1)

	// Go through the string in reverse order and replace all variables with their value
	expandedStr := str
	for i := len(indices) - 1; i >= 0; i-- {
		match := indices[i]
		// match[0] is the start index of the whole match
		startIndex := match[0]
		// match[1] is the end index of the whole match (exclusive)
		endIndex := match[1]
		// match[2] is start index of group
		startIndexGroup := match[2]
		// match[3] is end index of group (exclusive)
		endIndexGroup := match[3]

		varName := str[startIndexGroup:endIndexGroup]
		expandedStr = expandedStr[:startIndex] + vars[varName] + expandedStr[endIndex:]
	}

	return expandedStr
}
