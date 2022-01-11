package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/tb/errkind"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Provider struct {
	// client is the underlying S3 client to use.
	// It is lazily initialized the first time it is used.
	client *s3.Client
}

// init initializes the s3Provider by creating an S3 client and caching it.
func (p *s3Provider) init(ctx context.Context, op errors.Op) error {
	if p.client != nil {
		return nil
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.AWS,
			Reason: "failed to load AWS configuration",
			Op:     op,
		})
	}
	p.client = s3.NewFromConfig(cfg)
	return nil
}

func (p *s3Provider) GetObject(ctx context.Context, bucket string, key string) (io.ReadCloser, error) {
	const op = errors.Op("storage.s3Provider.GetObject")
	if err := p.init(ctx, op); err != nil {
		return nil, err
	}
	getObjectOutput, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.Meta{
			Kind:   errkind.AWS,
			Reason: fmt.Sprintf("failed to get object from S3: %s/%s", bucket, key),
			Op:     op,
		})
	}
	return getObjectOutput.Body, nil
}

func (p *s3Provider) ListObjectKeysByPrefix(ctx context.Context, bucket string, prefix string) ([]string, error) {
	const op = errors.Op("storage.s3Provider.ListObjectKeysByPrefix")
	if err := p.init(ctx, op); err != nil {
		return nil, err
	}
	listObjectsOutput, err := p.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:     aws.String(bucket),
		Prefix:     aws.String(prefix),
		StartAfter: aws.String(prefix + "/"),
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.Meta{
			Kind:   errkind.AWS,
			Reason: fmt.Sprintf("failed to list object in S3 bucket %s with prefix %s", bucket, prefix),
			Op:     op,
		})
	}
	// Parse response into an array of keys
	keys := make([]string, len(listObjectsOutput.Contents))
	for i, obj := range listObjectsOutput.Contents {
		keys[i] = *obj.Key
	}
	return keys, nil
}
