package cmd

import (
	"fmt"

	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/spinner"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/docker/registry"
	"github.com/spf13/cobra"
)

type imagesOptions struct {
	max            int
	dockerRegistry string
}

var imagesOpts imagesOptions

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
		service, err := config.LoadedServices().Get(serviceName)
		if err != nil {
			fatal.ExitErrf(err, "%s is not a valid service\n. Try running `tb list` to see available services\n", serviceName)
		}
		if service.Remote.Image == "" {
			fatal.Exitf("%s is not available from a remote docker registry\n", serviceName)
		}

		dockerRegistry, err := registry.GetRegistry(imagesOpts.dockerRegistry)
		if err != nil {
			fatal.ExitErrf(err, "Failed getting docker registry %s", imagesOpts.dockerRegistry)
		}

		s := spinner.New(
			spinner.WithStartMessage("☐ Fetching images for "+serviceName),
			spinner.WithStopMessage("☑ Finished fetching images for "+serviceName),
		)
		s.Start()

		imgs, err := dockerRegistry.FetchRepoImages(service.Remote.Image, imagesOpts.max)
		if err != nil {
			fatal.ExitErr(err, "Failed to fetch docker images")
		}
		s.Stop()

		for _, img := range imgs {
			fmt.Println(img.PushedAt, img.Tags)
		}
	},
}

func init() {
	rootCmd.AddCommand(imagesCmd)
	imagesCmd.Flags().IntVarP(&imagesOpts.max, "max", "m", 10, "maximum results to display")
	imagesCmd.Flags().StringVarP(&imagesOpts.dockerRegistry, "docker-registry", "r", "ecr", "type of docker registry, valid values: ecr")
}
