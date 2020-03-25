package storage

import "github.com/pkg/errors"

type StorageProvider interface {
	DownloadObject(bucket string, objKey string, dstPath string) error
	ListObjectKeysByPrefix(bucket string, objKeyPrefix string) ([]string, error)
}

func GetProvider(providerName string) (StorageProvider, error) {
	var provider StorageProvider
	switch providerName {
	case "s3":
		provider = S3StorageProvider{}
	default:
		return nil, errors.Errorf("Invalid storage provider %s", providerName)
	}

	return provider, nil
}
