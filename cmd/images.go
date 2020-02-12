package cmd

import (
	"fmt"

	"github.com/TouchBistro/tb/awsecr"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/fatal"
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
		if _, ok := config.Services()[serviceName]; !ok {
			fatal.Exitf("%s is not a valid service\n. Try running `tb list` to see available services\n", serviceName)
		}

		fmt.Printf("Fetching images for %s:\n", serviceName)
		images, err := awsecr.FetchRepoImages(serviceName, max)
		if err != nil {
			fatal.ExitErrf(err, "failed load images for service %s", serviceName)
		}

		for _, img := range images {
			fmt.Println(img.ImagePushedAt, img.ImageTags)
		}
	},
}

func init() {
	rootCmd.AddCommand(imagesCmd)
	imagesCmd.Flags().IntVarP(&max, "max", "m", 10, "maximum results to display")
}
