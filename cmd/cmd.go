package cmd

import "github.com/spf13/cobra"

func Commands() []*cobra.Command {
	return []*cobra.Command{
		cloneCmd,
		dbCmd,
		execCmd,
		imagesCmd,
		listCmd,
		logsCmd,
		nukeCmd,
		upCmd,
	}
}
