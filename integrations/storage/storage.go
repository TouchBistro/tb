// Package storage provides functionality for working with blob storage providers.
package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/tb/errkind"
)

// Provider represents all functionality provided by a blob storage provider.
type Provider interface {
	// GetObject retrieves a single object from the storage provider.
	// It is the caller's responsibility to close the returned io.ReadCloser
	// once they are done reading from it.
	GetObject(ctx context.Context, bucket string, key string) (io.ReadCloser, error)
	// ListObjectKeysByPrefix returns a list of keys for all objects in the given bucket
	// that start with the give prefix.
	ListObjectKeysByPrefix(ctx context.Context, bucket string, prefix string) ([]string, error)
}

// NewProvider returns a new Provider based on the given providerName.
func NewProvider(providerName string) (Provider, error) {
	switch providerName {
	case "s3":
		return &s3Provider{}, nil
	default:
		return nil, errors.New(
			errkind.Invalid,
			fmt.Sprintf("unknown storage provider %s", providerName),
			"storage.NewProvider",
		)
	}
}
