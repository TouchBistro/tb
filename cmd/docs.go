package cmd

import (
	"github.com/TouchBistro/goutils/fatal"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var docsCmd = &cobra.Command{
	Hidden: true,
	Use:    "docs",
	Args:   cobra.NoArgs,
	Short:  "Generate documentation for all the commands",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("Generating markdown documentation...")
		err := doc.GenMarkdownTree(rootCmd, "./docs")
		if err != nil {
			fatal.ExitErr(err, "Failed to generate documentation.")
		}
		log.Info("done...")
	},
}

func init() {
	rootCmd.AddCommand(docsCmd)
}
