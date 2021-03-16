package storage

import (
	"context"

	"github.com/TouchBistro/goutils/file"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
)

type S3StorageProvider struct{}

func (s S3StorageProvider) ListObjectKeysByPrefix(bucket, objKeyPrefix string) ([]string, error) {
	// Set up AWS Env Vars
	ctx := context.Background()
	conf, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load AWS configuration")
	}

	// Make S3 Request
	client := s3.NewFromConfig(conf)
	listObjectsOutput, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:     aws.String(bucket),
		Prefix:     aws.String(objKeyPrefix),
		StartAfter: aws.String(objKeyPrefix + "/"),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed listing objects in S3")
	}

	// Parse response into an array of keys
	keys := make([]string, len(listObjectsOutput.Contents))
	for i, obj := range listObjectsOutput.Contents {
		keys[i] = *obj.Key
	}
	return keys, nil
}

func (s S3StorageProvider) DownloadObject(bucket, objKey, dstPath string) error {
	// Set up AWS Env Vars
	ctx := context.Background()
	conf, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to load AWS configuration")
	}

	// Request file from S3
	client := s3.NewFromConfig(conf)
	getObjectOutput, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objKey),
	})
	if err != nil {
		return errors.Wrap(err, "failed to get object from S3")
	}
	defer getObjectOutput.Body.Close()

	// Download to a local file.
	_, err = file.Download(dstPath, getObjectOutput.Body)
	if err != nil {
		return errors.Wrapf(err, "failed downloading file to %s", dstPath)
	}
	return nil
}
