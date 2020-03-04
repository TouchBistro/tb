package cmd

import (
	"fmt"
	"strings"

	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/git"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone [service...]",
	Short: "Clone a tb service",
	Long: `Clone any service in service.yml that has repo set to true

	Examples:
		tb clone venue-admin-frontend`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := args[0]

		s, err := config.LoadedServices().Get(serviceName)
		if err != nil {
			fatal.ExitErrf(err, "%s is not a valid service.\nTry running `tb list` to see available services\n", serviceName)
		}

		if !s.HasGitRepo() {
			fatal.Exitf("%s does not have a repo or is a third-party repo\n", serviceName)
		}

		repoPath := fmt.Sprintf("./%s", strings.Split(s.GitRepo, "/")[1])
		err = git.Clone(s.GitRepo, repoPath)
		if err != nil {
			fatal.ExitErr(err, "Could not run git clone command.")
		}

		log.Infof("☑ cloning of %s was successful", serviceName)
	},
}

func init() {
	rootCmd.AddCommand(cloneCmd)
}
