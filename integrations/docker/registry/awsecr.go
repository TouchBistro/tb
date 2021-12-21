package registry

import (
	"context"
	"regexp"
	"sort"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/tb/errkind"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

type ecrDockerRegistry struct{}

func (ecrDockerRegistry) FetchRepoImages(ctx context.Context, image string, limit int) ([]ImageDetail, error) {
	const op = errors.Op("registry.ecrDockerRegistry.FetchRepoImages")
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, errors.Wrap(err, errors.Meta{
			Kind:   errkind.AWS,
			Reason: "failed to load AWS configuration",
			Op:     op,
		})
	}
	ecrClient := ecr.NewFromConfig(cfg)

	// Need to strip ECR registry prefix from the image to get repo name
	// i.e. <aws_ccount_id>.dkr.ecr.<region>.amazonaws.com/<repo>
	regex := regexp.MustCompile(`.+\.dkr\.ecr\..+\.amazonaws\.com\/(.+)`)
	repoName := regex.FindStringSubmatch(image)[1]

	// Unforunately there's no way to get the latest images from ECR
	// it just seems to return them in a random order
	// We therefore need to fetch all the available images
	// and sort them ourselves to get the newest ones
	const max = 1000
	describeImagesInput := &ecr.DescribeImagesInput{
		RepositoryName: aws.String(repoName),
		MaxResults:     aws.Int32(int32(max)),
	}
	var images []types.ImageDetail
	for {
		describeImagesOutput, err := ecrClient.DescribeImages(ctx, describeImagesInput)
		if err != nil {
			return nil, errors.Wrap(err, errors.Meta{
				Kind:   errkind.AWS,
				Reason: "failed to load images from ECR repo",
				Op:     op,
			})
		}
		images = append(images, describeImagesOutput.ImageDetails...)
		if describeImagesOutput.NextToken == nil {
			break
		}
		describeImagesInput.NextToken = describeImagesOutput.NextToken
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
