package cmd

import (
	"fmt"

	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/spinner"
	"github.com/TouchBistro/tb/awsecr"
	"github.com/TouchBistro/tb/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	max int
)

var imagesCmd = &cobra.Command{
	Use:     "images",
	Aliases: []string{"img"},
	Args:    cobra.ExactArgs(1),
	Short:   "List latest available images for a service",
	Long: `List latest available images for a service.
	
Examples:
- List the last 10 images available for venue-core-service in the container registry
	tb images venue-core-service --max 10
`,
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := args[0]
		s, ok := config.Services()[serviceName]
		if !ok {
			fatal.Exitf("%s is not a valid service\n. Try running `tb list` to see available services\n", serviceName)
		} else if s.Remote.Image == "" {
			fatal.Exitf("%s is not available from a remote docker registry\n", serviceName)
		}

		log.Infof("☐ Fetching images for %s:", serviceName)
		successCh := make(chan string)
		failedCh := make(chan error)
		var images []ecr.ImageDetail

		// Do it for the spinner!
		go func() {
			imgs, err := awsecr.FetchRepoImages(serviceName, max)
			if err != nil {
				failedCh <- err
				return
			}

			// This is only being assigned in this one goroutine so locking isn't needed
			images = imgs
			successCh <- serviceName
		}()

		spinner.SpinnerWait(successCh, failedCh, "☑ Finished fetching images for %s\n", "failed loading images", 1)

		for _, img := range images {
			fmt.Println(img.ImagePushedAt, img.ImageTags)
		}
	},
}

func init() {
	rootCmd.AddCommand(imagesCmd)
	imagesCmd.Flags().IntVarP(&max, "max", "m", 10, "maximum results to display")
}
