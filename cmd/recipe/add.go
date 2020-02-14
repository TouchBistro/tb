package recipe

import (
	"github.com/TouchBistro/goutils/color"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <recipe-name>",
	Args:  cobra.ExactArgs(1),
	Short: "Adds the given recipe to tb",
	Long: `Adds the given recipe to tb.
	
Example:
- adds the recipe named 'tb-recipe-services'
	tb recipe add tb-recipe-services`,
	Run: func(cmd *cobra.Command, args []string) {
		recipeName := args[0]
		log.Infof(color.Cyan("☐ Adding recipe %s..."), recipeName)

		err := config.AddRecipe(recipeName)
		if err == config.ErrRecipeExists {
			log.Infof("recipe %s has already been added", recipeName)
		}

		if err != nil {
			fatal.ExitErrf(err, "failed to add recipe %s", recipeName)
		}

		log.Infof(color.Green("☑ Successfully added recipe %s"), recipeName)
	},
}

func init() {
	recipeCmd.AddCommand(addCmd)
}
