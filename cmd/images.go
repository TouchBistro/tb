package cmd

import (
	"fmt"
	"github.com/TouchBistro/tb/awsecr"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/fatal"
	"github.com/spf13/cobra"
)

var (
	serviceName string
	max      int64
)

var imagesCmd = &cobra.Command{
	Use:     "images",
	Aliases: []string{"img"},
	Args:    cobra.NoArgs,
	Short:   "List latest available images for a service",
	Long: `List latest available images for a service.
	
Examples:
- List the last 10 images available for venue-core-service in the container registry
	tb images --service venue-core-service --max 10
`,
	Run: func(cmd *cobra.Command, args []string) {
		if _, ok := config.Services()[serviceName]; !ok {
			fatal.Exitf("%s is not a valid service\n. Try running `tb list` to see available services\n", serviceName)
		}

		fmt.Println("Fetching images for "+serviceName+":")
		listImages(serviceName, max)
	},
}

func listImages(serviceName string, max int64) {
	images, err := awsecr.FetchRepoImages(serviceName)

	if err != nil {
		fatal.ExitErr(err, "☒ failed load images for service "+serviceName)
	}

	for i := 0; i < int(max); i++ {
		fmt.Println(images[i].ImagePushedAt, images[i].ImageTags)
	}
}

func init() {
	rootCmd.AddCommand(imagesCmd)
	imagesCmd.Flags().StringVarP(&serviceName, "service", "s", "", "name of service")
	imagesCmd.Flags().Int64VarP(&max, "max", "m", 10, "maximum results to display")
	imagesCmd.MarkFlagRequired("service")
}
