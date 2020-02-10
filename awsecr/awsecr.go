package awsecr

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/pkg/errors"
	"sort"
)

const Limit = int64(1000)

type ImgDetail []ecr.ImageDetail

func FetchRepoImages(repoName string) ([]ecr.ImageDetail, error) {
	conf, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, errors.Wrap(err, "☒ failed to load default aws config")
	}

	maxResult := Limit

	input := ecr.DescribeImagesInput{
		RepositoryName: &repoName,
		MaxResults: &maxResult,
	}

	images := ImgDetail{}

	client := ecr.New(conf)
	ctx := context.Background()

	for {
		req := client.DescribeImagesRequest(&input)
		res, err := req.Send(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "☒ failed to load images for repo")
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

	return images, nil
}
