package registry

import (
	"context"
	"regexp"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/pkg/errors"
)

type ECRDockerRegistry struct{}

func (_ ECRDockerRegistry) FetchRepoImages(ecrImage string, limit int) ([]ImageDetail, error) {
	conf, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load default aws config")
	}

	// Need to strip ECR registry prefix from the image to get repo name
	// i.e. <aws_ccount_id>.dkr.ecr.<region>.amazonaws.com/<repo>
	regex := regexp.MustCompile(`.+\.dkr\.ecr\..+\.amazonaws\.com\/(.+)`)
	repoName := regex.FindStringSubmatch(ecrImage)[1]

	// Unforunately there's no way to get the latest images from ECR
	// it just seems to return them in a random order
	// We therefore need to fetch all the available images
	// and sort them ourselves to get the newest ones
	const max = 1000
	input := ecr.DescribeImagesInput{
		RepositoryName: aws.String(repoName),
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

	images = images[:limit]

	imageDetails := make([]ImageDetail, len(images))
	for i, img := range images {
		imageDetails[i] = ImageDetail{
			PushedAt: img.ImagePushedAt,
			Tags:     img.ImageTags,
		}
	}

	return imageDetails, nil
}
