package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of tb",
	Args:  cobra.NoArgs,
	Long:  `All software has versions. This is TB's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("XTREME BETA 0.0.0")
	},
}
