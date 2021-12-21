package commands

import (
	"fmt"
	"strings"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/integrations/git"
	"github.com/TouchBistro/tb/resource"

	"github.com/spf13/cobra"
)

func newCloneCommand(c *cli.Container) *cobra.Command {
	return &cobra.Command{
		Use:        "clone [service]",
		Deprecated: "it will be removed soon",
		Short:      "Clone a tb service",
		Long: `Clone any service in service.yml that has repo set to true

		Examples:
			tb clone venue-admin-frontend`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := c.Engine.ResolveService(args[0])
			if errors.Is(err, resource.ErrNotFound) {
				return &cli.ExitError{
					Message: "Try running `tb list` to see available services",
					Err:     err,
				}
			} else if err != nil {
				return err
			}
			if !s.HasGitRepo() {
				return &cli.ExitError{
					Message: fmt.Sprintf("%s does not have a repo", s.FullName()),
				}
			}

			ctx := progress.ContextWithTracker(cmd.Context(), c.Tracker)
			repoPath := fmt.Sprintf("./%s", strings.Split(s.GitRepo.Name, "/")[1])
			err = git.New().Clone(ctx, s.GitRepo.Name, repoPath)
			if err != nil {
				return err
			}
			c.Tracker.Infof("Cloning of %s was successful", s.FullName())
			return err
		},
	}
}
