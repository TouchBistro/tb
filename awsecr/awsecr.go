package awsecr

import (
	"context"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/pkg/errors"
)

const max = 1000

func FetchRepoImages(repoName string, limit int) ([]ecr.ImageDetail, error) {
	conf, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load default aws config")
	}

	input := ecr.DescribeImagesInput{
		RepositoryName: &repoName,
		MaxResults:     aws.Int64(int64(max)),
	}

	images := []ecr.ImageDetail{}

	client := ecr.New(conf)
	ctx := context.Background()

	for {
		req := client.DescribeImagesRequest(&input)
		res, err := req.Send(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load images for repo")
		}

		images = append(images, res.ImageDetails...)
		if res.NextToken == nil {
			break
		}

		input.NextToken = res.NextToken
	}

	sort.Slice(images, func(i, j int) bool {
		return images[i].ImagePushedAt.After(*images[j].ImagePushedAt)
	})

	return images[:limit], nil
}
