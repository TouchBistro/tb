package util

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func FileOrDirExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
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

func CopyFile(srcPath, destPath string) error {
	srcStat, err := os.Stat(srcPath)
	if err != nil {
		return errors.Wrapf(err, "failed to get info of %s", srcPath)
	}

	if !srcStat.Mode().IsRegular() {
		return errors.New(fmt.Sprintf("%s is not a file", srcPath))
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return errors.Wrapf(err, "failed to open file %s", srcPath)
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(
		destPath,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		srcStat.Mode(),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to open file %s", destPath)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return errors.Wrapf(err, "failed to copy %s to %s", srcPath, destPath)
}

func CopyDirContents(srcPath, destPath string) error {
	stat, err := os.Stat(srcPath)
	if err != nil {
		return errors.Wrapf(err, "failed to get info of %s", srcPath)
	}

	if !stat.IsDir() {
		return errors.New(fmt.Sprintf("%s is not a directory", srcPath))
	}

	err = os.MkdirAll(destPath, stat.Mode())
	if err != nil {
		return errors.Wrapf(err, "failed to create missing directories for destPath %s", destPath)
	}

	contents, err := ioutil.ReadDir(srcPath)
	if err != nil {
		return errors.Wrapf(err, "failed to read contents of %s", srcPath)
	}

	for _, item := range contents {
		srcItemPath := fmt.Sprintf("%s/%s", srcPath, item.Name())
		destItemPath := fmt.Sprintf("%s/%s", destPath, item.Name())

		if !item.IsDir() {
			err = CopyFile(srcItemPath, destItemPath)
			if err != nil {
				return errors.Wrapf(err, "failed to copy file %s", srcItemPath)
			}

			continue
		}

		err = CopyDirContents(srcItemPath, destItemPath)
		if err != nil {
			return errors.Wrapf(err, "failed to copy directory %s", srcItemPath)
		}
	}

	return nil
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
