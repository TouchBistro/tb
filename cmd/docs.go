package cmd

import (
	"fmt"
	"os/exec"

	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/config"
	"github.com/spf13/cobra"
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Args:  cobra.MinimumNArgs(1),
	Short: "Opens link to API docs for a given service",
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := args[0]
		service, err := config.LoadedServices().Get(serviceName)
		if err != nil {
			fatal.ExitErrf(err, "%s is not a valid service.\nTry running `tb list` to see available services\n", serviceName)
		}

		if !service.HasGitRepo() {
			fatal.Exitf("%s does not have a repo or is a third-party repo\n", serviceName)
		}

		docsURL := service.EnvVars["API_DOCS_URL"]
		if docsURL == "" {
			fatal.Exitf("API_DOCS_URL environment variable not found for service %s\n", serviceName)
		}

		fmt.Printf("Opening docs for %s...\n", serviceName)
		openDocs(docsURL)
	},
}

func openDocs(url string) {
	cmd := exec.Command("open", url)
	if err := cmd.Run(); err != nil {
		fatal.ExitErrf(err, "failed to open docs at %s\n", url)
	}
}

func init() {
	rootCmd.AddCommand(docsCmd)
}
