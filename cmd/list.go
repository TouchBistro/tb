package cmd

import (
	"bufio"
	"context"
	"fmt"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/fatal"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/spf13/cobra"
	"os"
	"sort"
)

var (
	shouldListServices        bool
	shouldListPlaylists       bool
	shouldListCustomPlaylists bool
	shouldListECRImages       bool
	isTreeMode                bool
	repoName                  string
	maxResult                 int64
)

type imgDetail []ecr.ImageDetail

func (img imgDetail) Len() int {
	return len(img)
}

func (img imgDetail) Less(i, j int) bool {
	return img[i].ImagePushedAt.Before(*img[j].ImagePushedAt)
}

func (img imgDetail) Swap(i, j int) {
	img[i], img[j] = img[j], img[i]
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Args:    cobra.NoArgs,
	Short:   "Lists all available services",
	Run: func(cmd *cobra.Command, args []string) {
		// If no flags provided show everything
		if !shouldListServices &&
			!shouldListPlaylists &&
			!shouldListCustomPlaylists &&
			!shouldListECRImages {
			shouldListServices = true
			shouldListPlaylists = true
			shouldListCustomPlaylists = true
		}

		if shouldListServices {
			fmt.Println("Services:")
			listServices(config.Services())
		}

		if shouldListPlaylists {
			fmt.Println("Playlists:")
			listPlaylists(config.Playlists(), isTreeMode)
		}

		if shouldListCustomPlaylists {
			fmt.Println("Custom Playlists:")
			listPlaylists(config.TBRC().Playlists, isTreeMode)
		}

		if shouldListECRImages && len(repoName) < 1 {
			fatal.Exit("ecr repo name is required")
		} else {
			fmt.Println("ECR Images:")
			listECRImages(repoName, maxResult)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&shouldListServices, "services", "s", false, "list services")
	listCmd.Flags().BoolVarP(&shouldListPlaylists, "playlists", "p", false, "list playlists")
	listCmd.Flags().BoolVarP(&shouldListCustomPlaylists, "custom-playlists", "c", false, "list custom playlists")
	listCmd.Flags().BoolVarP(&shouldListECRImages, "ecr-images", "e", false, "list ecr images")
	listCmd.Flags().BoolVarP(&isTreeMode, "tree", "t", false, "tree mode, show playlist services")
	listCmd.Flags().StringVarP(&repoName, "repo", "r", "", "ecr repo name")
	listCmd.Flags().Int64VarP(&maxResult, "max", "m", 10, "ecr image list max result")
}

func listServices(services config.ServiceMap) {
	names := make([]string, len(services))
	i := 0
	for name := range services {
		names[i] = name
		i++
	}

	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("  - %s\n", name)
	}
}

func listPlaylists(playlists map[string]config.Playlist, tree bool) {
	names := make([]string, len(playlists))
	i := 0
	for name := range playlists {
		names[i] = name
		i++
	}

	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("  - %s\n", name)
		if !tree {
			continue
		}
		list, err := config.GetPlaylist(name, make(map[string]bool))
		if err != nil {
			fatal.ExitErr(err, "☒ failed resolving service playlist")
		}
		for _, s := range list {
			fmt.Printf("    - %s\n", s)
		}
	}
}

func fetchImages(client *ecr.Client, input ecr.DescribeImagesInput, ctx context.Context) {
	var enterKeyCode int = 1

	req := client.DescribeImagesRequest(&input)

	res, err := req.Send(ctx)
	if err != nil {
		fatal.ExitErr(err, "☒ failed load ecr images")
	}

	sortedImages := make(imgDetail, 0, len(res.ImageDetails))

	for _, img := range  res.ImageDetails {
		sortedImages = append(sortedImages, img)
	}

	sort.Sort(sortedImages)

	for _, img := range sortedImages {
		fmt.Println(img.ImagePushedAt, img.ImageTags)
	}

	if res.NextToken != nil {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("Hit enter to load more:")

		_, s, err := reader.ReadRune()

		if err != nil {
			fatal.ExitErr(err, "☒ failed to read input")
		}

		if s == enterKeyCode {
			input.NextToken = res.NextToken
			fetchImages(client, input, ctx)
		}
	}
}

func listECRImages(repoName string, maxResult int64) {
	var input ecr.DescribeImagesInput
	var ctx = context.Background()

	input.RepositoryName = &repoName
	input.MaxResults = &maxResult

	conf, err := external.LoadDefaultAWSConfig()
	client := ecr.New(conf)

	if err != nil {
		fatal.ExitErr(err, "☒ failed load ecr images")
	}

	fetchImages(client, input, ctx)
}
