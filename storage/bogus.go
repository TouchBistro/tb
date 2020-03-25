package storage

import (
	"github.com/pkg/errors"
)

// why the empty struct? Just a nominal type??
// NM - i extend it below with the fancy syntax
type BogusStorageProvider struct{}

func (s BogusStorageProvider) Name() string {
	return "bogus"
}

func (s BogusStorageProvider) ListObjectKeysByPrefix(bucket, objKeyPrefix string) ([]string, error) {
	return nil, errors.New("bogus")
}

func (s BogusStorageProvider) DownloadObject(bucket, objKey, dstPath string) error {
	return errors.New("bogus")
}
