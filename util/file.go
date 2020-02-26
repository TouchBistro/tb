package util

import (
	"bufio"
	"io"
	"os"
	"path/filepath"

	"github.com/TouchBistro/goutils/command"
	"github.com/TouchBistro/goutils/file"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func Untar(src string, cleanup bool) error {
	if !file.FileOrDirExists(src) {
		return errors.New(src + " does not exist")
	}

	flags := ""
	ext := filepath.Ext(src)
	if ext == ".tar" {
		flags = "-xf"
	} else if ext == ".tgz" {
		flags = "-xzf"
	} else {
		return errors.New(src + " does not end in .tar or .tgz")
	}

	err := command.Exec("tar", []string{flags, src, "-C", filepath.Dir(src)}, "untar")
	if err != nil {
		return errors.Wrapf(err, "failed to extract %s in its parent directory", src)
	}

	if cleanup {
		err := os.Remove(src)
		if err != nil {
			return errors.Wrapf(err, "failed to delete %s after extracting", src)
		}
	}

	return nil
}

func DownloadFile(downloadPath string, r io.Reader) (int64, error) {
	// Check if file exists
	downloadDir := filepath.Dir(downloadPath)
	if !file.FileOrDirExists(downloadDir) {
		err := os.MkdirAll(downloadDir, 0755)
		if err != nil {
			return 0, errors.Wrapf(err, "could not create directory %s", downloadDir)
		}
	}

	// Write payload to target dir
	f, err := os.Create(downloadPath)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to create file %s", downloadPath)
	}
	w := bufio.NewWriter(f)
	nBytes, err := io.Copy(w, r)
	if err != nil {
		return 0, errors.Wrap(err, "failed writing build to file")
	}
	w.Flush()

	return nBytes, nil
}

func ReadYaml(path string, val interface{}) error {
	file, err := os.Open(path)
	if err != nil {
		return errors.Wrapf(err, "failed to open file %s", path)
	}
	defer file.Close()

	err = yaml.NewDecoder(file).Decode(val)
	return errors.Wrapf(err, "failed to decode yaml file %s", path)
}
