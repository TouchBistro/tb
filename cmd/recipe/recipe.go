package recipe

import (
	"github.com/spf13/cobra"
)

var recipeCmd = &cobra.Command{
	Use:   "recipe",
	Short: "tb recipe manages recipes from the command line",
}

func Recipe() *cobra.Command {
	return recipeCmd
}
