package commands

import (
	"fmt"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/tb/cli"
	dockerregistry "github.com/TouchBistro/tb/integrations/docker/registry"
	"github.com/TouchBistro/tb/resource"
	"github.com/spf13/cobra"
)

func newImagesCommand(c *cli.Container) *cobra.Command {
	var imagesOpts struct {
		max            int
		dockerRegistry string
	}
	imagesCmd := &cobra.Command{
		Use:        "images",
		Deprecated: "it will be removed soon",
		Aliases:    []string{"img"},
		Args:       cobra.ExactArgs(1),
		Short:      "List latest available images for a service",
		Long: `List latest available images for a service.

	Examples:
	- List the last 10 images available for venue-core-service in the container registry
		tb images venue-core-service --max 10
	`,
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := c.Engine.ResolveService(args[0])
			if errors.Is(err, resource.ErrNotFound) {
				return &cli.ExitError{
					Message: "Try running `tb list` to see available services",
					Err:     err,
				}
			} else if err != nil {
				return err
			}
			if service.Remote.Image == "" {
				return &cli.ExitError{
					Message: fmt.Sprintf("%s is not available from a remote docker registry", service.FullName()),
				}
			}

			dockerRegistry, err := dockerregistry.Get(imagesOpts.dockerRegistry)
			if err != nil {
				return err
			}

			c.Tracker.Start(fmt.Sprintf("Fetching images for %s", service.FullName()), 0)
			imgs, err := dockerRegistry.FetchRepoImages(cmd.Context(), service.Remote.Image, imagesOpts.max)
			c.Tracker.Stop()
			if err != nil {
				return &cli.ExitError{
					Message: "Failed to fetch docker images",
					Err:     err,
				}
			}
			for _, img := range imgs {
				fmt.Println(img.PushedAt, img.Tags)
			}
			return nil
		},
	}
	imagesCmd.Flags().IntVarP(&imagesOpts.max, "max", "m", 10, "maximum results to display")
	imagesCmd.Flags().StringVarP(&imagesOpts.dockerRegistry, "docker-registry", "r", "ecr", "type of docker registry, valid values: ecr")
	return imagesCmd
}
