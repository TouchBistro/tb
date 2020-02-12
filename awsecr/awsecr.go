package awsecr

import (
	"context"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/pkg/errors"
)

func FetchRepoImages(repoName string, limit int) ([]ecr.ImageDetail, error) {
	conf, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load default aws config")
	}

	input := ecr.DescribeImagesInput{
		RepositoryName: &repoName,
		MaxResults:     aws.Int64(int64(limit)),
	}

	images := []ecr.ImageDetail{}

	client := ecr.New(conf)
	ctx := context.Background()
	count := 0

	for {
		req := client.DescribeImagesRequest(&input)
		res, err := req.Send(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load images for repo")
		}

		images = append(images, res.ImageDetails...)

		// MaxResults set the limit for the first page
		// but more values can be returned by the NextToken
		// Stop if we already got the amount of images we need
		count += len(res.ImageDetails)
		if count >= limit || res.NextToken == nil {
			break
		}

		input.NextToken = res.NextToken
	}

	sort.Slice(images, func(i, j int) bool {
		return images[i].ImagePushedAt.After(*images[j].ImagePushedAt)
	})

	return images, nil
}
