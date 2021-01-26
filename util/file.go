package util

import (
	"os"
	"path/filepath"

	"github.com/TouchBistro/goutils/command"
	"github.com/TouchBistro/goutils/file"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func Untar(src string, cleanup bool) error {
	if !file.Exists(src) {
		return errors.Errorf("%s does not exist", src)
	}

	flags := ""
	ext := filepath.Ext(src)
	if ext == ".tar" {
		flags = "-xf"
	} else if ext == ".tgz" {
		flags = "-xzf"
	} else {
		return errors.Errorf("%s does not end in .tar or .tgz", src)
	}

	// TODO(@cszatmary): Scope archive/tar and compress/gzip from the stdlib to use
	// instead of shelling to tar
	w := log.WithField("id", "untar").WriterLevel(log.DebugLevel)
	defer w.Close()
	cmd := command.New(command.WithStdout(w), command.WithStderr(w))
	err := cmd.Exec("tar", flags, src, "-C", filepath.Dir(src))
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
