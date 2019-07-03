package cmd

import (
	"fmt"
	"log"

	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/util"
	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "clones repositories from config to the current working dir",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Checking repos...")

		services := *config.All()
		for _, s := range services {
			path := fmt.Sprintf("./%s", s.Name)
			if !s.Repo || util.FileOrDirExists(path) {
				continue
			}

			fmt.Printf("%s is missing. cloning...\n", s.Name)

			repoURL := fmt.Sprintf("git@github.com:TouchBistro/%s.git", s.Name)
			err := util.ExecStdoutStderr("git", "clone", repoURL)
			if err != nil {
				log.Fatal(err)
			}
		}
		log.Println("...done")
	},
}

func init() {
	RootCmd.AddCommand(cloneCmd)
}
