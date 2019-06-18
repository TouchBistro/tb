package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/TouchBistro/tb/config"
)

func exists(filePath string) bool {
  if _, err := os.Stat(filePath); os.IsNotExist(err) {
    return false
  }
  return true
}

var cloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "clones repositories from config to the current working dir",
	Run: func(cmd *cobra.Command, args []string) {
	  log.Println("Checking repos...")

    services := *config.All()
    for _, s := range services {
      path := fmt.Sprintf("./%s", s.Name)
      if !s.Repo || exists(path) {
        continue
      }

      fmt.Printf("%s is missing. cloning...\n", s.Name)

      cmdStr := fmt.Sprintf("clone git@github.com:TouchBistro/%s.git", s.Name)
      cmd := exec.Command("git", strings.Fields(cmdStr)...)
      cmd.Stdout = os.Stdout
	    cmd.Stderr = os.Stderr

	    err := cmd.Run()
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
