package cmd

import (
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "tb",
	Short: "tb is a CLI for running TouchBistro services on a development machine",
}
