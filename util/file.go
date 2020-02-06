package util

import (
	"bufio"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func Untar(src string, cleanup bool) error {
	if !FileOrDirExists(src) {
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

	label := "untar"
	err := Exec(label, "tar", flags, src, "-C", filepath.Dir(src))
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
	if !FileOrDirExists(downloadDir) {
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

func FileOrDirExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func DirLen(path string) (int, error) {
	dir, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	list, err := dir.Readdirnames(0)
	return len(list), err
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
		return errors.Errorf("%s is not a file", srcPath)
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
		return errors.Errorf("%s is not a directory", srcPath)
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
		srcItemPath := filepath.Join(srcPath, item.Name())
		destItemPath := filepath.Join(destPath, item.Name())

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

func DirSize(path string) (int64, error) {
	if !FileOrDirExists(path) {
		return 0, nil
	}

	var size int64
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrapf(err, "failed to walk %s", path)
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}
