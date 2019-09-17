package util

import (
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func FileOrDirExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func MkdirFullpath(path string) error {
	parent := strings.Join(strings.Split(path, "/")[:len(strings.Split(path, "/"))-1], "/")
	_, err := os.Stat(parent)
	if os.IsNotExist(err) {
		err = MkdirFullpath(parent)
		if err != nil {
			return nil
		}
	}
	err = os.Mkdir(path, 0766)
	return err
}

func AppendLineToFile(path string, line string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to open file %s", path)
	}
	defer f.Close()

	_, err = f.WriteString(line + "\n")
	return errors.Wrapf(err, "failed to write line %s to file %s", path, line)
}

func CreateFile(path string, content string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to open file %s", path)
	}
	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		return errors.Wrapf(err, "failed to write content to file %s", path)
	}

	err = f.Sync()
	return errors.Wrapf(err, "failed to commit write to disk writing to %s", path)
}

func ReadYaml(path string, val interface{}) error {
	file, err := os.Open(path)
	if err != nil {
		return errors.Wrapf(err, "failed to open file %s", path)
	}
	defer file.Close()

	err = DecodeYaml(file, val)
	return errors.Wrapf(err, "failed to decode yaml file %s", path)
}

func DecodeYaml(r io.Reader, val interface{}) error {
	dec := yaml.NewDecoder(r)
	err := dec.Decode(val)

	return err
}
