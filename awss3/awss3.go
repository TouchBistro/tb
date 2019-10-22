package awss3

import (
	"context"
	"path/filepath"

	"github.com/TouchBistro/tb/util"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func ListObjectKeysByPrefix(bucket, objKeyPrefix string) ([]string, error) {
	// Set up AWS Env Vars
	conf, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load AWS configuration")
	}

	// Make S3 Request
	client := s3.New(conf)
	ctx := context.Background()
	req := client.ListObjectsV2Request(&s3.ListObjectsV2Input{
		Bucket:     aws.String(bucket),
		Prefix:     aws.String(objKeyPrefix),
		StartAfter: aws.String(objKeyPrefix),
	})
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed sending ListOBjectsV2Request to S3")
	}

	// Parse response into an array of keys
	keys := make([]string, len(resp.Contents))
	for i, obj := range resp.Contents {
		keys[i] = *obj.Key
	}

	return keys, nil
}

func DownloadObject(bucket, objKey, dstDir string) error {
	// Set up AWS Env Vars
	conf, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return errors.Wrap(err, "failed to load AWS configuration")
	}

	// Request file from S3
	client := s3.New(conf)
	ctx := context.Background()
	req := client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objKey),
	})
	resp, err := req.Send(ctx)
	if err != nil {
		return errors.Wrap(err, "failed sending GetObjectRequest to S3")
	}
	defer resp.Body.Close()

	// Download to a local file.
	dlPath := filepath.Join(dstDir, objKey)
	nBytes, err := util.DownloadFile(dlPath, resp.Body)
	if err != nil {
		return errors.Wrapf(err, "failed downloading file to %s", dlPath)
	}

	log.Debugf("Wrote %d bytes to %s successfully", nBytes, dlPath)

	return nil
}
