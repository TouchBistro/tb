package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "up",
	Short: "",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("harambe generation")
	},
}

func init() {
	RootCmd.AddCommand(addCmd)
}
