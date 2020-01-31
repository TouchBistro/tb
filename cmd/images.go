package cmd

import (
	"context"
	"fmt"
	"github.com/TouchBistro/tb/fatal"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sort"
)
var (
	repoName                  string
	maxResult                 int64
)

type ImgDetail []ecr.ImageDetail

var imagesCmd = &cobra.Command{
	Use:     "images",
	Aliases: []string{"im"},
	Args:    cobra.NoArgs,
	Short:   "Lists all images in a repo",
	Run: func(cmd *cobra.Command, args []string) {
		// If no flags provided show everything
		if len(repoName) < 1 {
			fatal.Exit("repository name is required")
		}

		fmt.Println("List Images:")
		listImages(repoName, maxResult)
	},
}

func init() {
	rootCmd.AddCommand(imagesCmd)
	imagesCmd.Flags().StringVarP(&repoName, "repo", "r", "", "repository name")
	imagesCmd.Flags().Int64VarP(&maxResult, "max", "m", 10, "image list max result")
}

func listImages(repoName string, maxResult int64) {
	images, err := FetchRepoImages(repoName)

	if err != nil {
		fatal.ExitErr(err, "☒ failed load ecr images")
	}

	sort.Slice(images, func (i, j int) bool {
		return images[i].ImagePushedAt.After(*images[j].ImagePushedAt)
	})

	for i := 0; i < int(maxResult) - 1; i++ {
		fmt.Println(images[i].ImagePushedAt, images[i].ImageTags)
	}
}

func FetchRepoImages(repoName string) ([]ecr.ImageDetail, error) {
	conf, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, errors.Wrap(err, "☒ failed to load default aws config")
	}

	maxResult := int64(1000)

	input := ecr.DescribeImagesInput{
		RepositoryName: &repoName,
		MaxResults: &maxResult,
	}

	images := ImgDetail{}

	client := ecr.New(conf)
	ctx := context.Background()

	if err != nil {
		return nil, errors.Wrap(err, "☒ failed to load fetch images")
	}

	for {
		req := client.DescribeImagesRequest(&input)
		res, err := req.Send(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "☒ failed to load fetch images")
		}

		images = append(images, res.ImageDetails...)

		if res.NextToken == nil {
			break
		}

		input.NextToken = res.NextToken
	}

	return images, nil
}