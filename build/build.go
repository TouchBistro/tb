package main

import (
	"fmt"
	"os"

	"github.com/TouchBistro/tb/cmd"
	"github.com/TouchBistro/tb/util"
	"github.com/spf13/cobra/doc"
)

func main() {
	rootCmd := cmd.Root()
	dir := "dist"
	if !util.FileOrDirExists(dir) {
		err := os.Mkdir(dir, 0755)
		if err != nil {
			util.FatalErr("Failed to create dist directory", err)
		}
	}

	zshCompPath := fmt.Sprintf("%s/_tb", dir)
	err := rootCmd.GenZshCompletionFile(zshCompPath)
	if err != nil {
		util.FatalErr("Failed to create zsh completions", err)
	}

	bashCompPath := fmt.Sprintf("%s/tb.bash", dir)
	err = rootCmd.GenBashCompletionFile(bashCompPath)
	if err != nil {
		util.FatalErr("Failed to create bash completions", err)
	}

	header := &doc.GenManHeader{
		Title:   "tb",
		Section: "1",
	}

	manDir := dir + "/man1"
	if !util.FileOrDirExists(manDir) {
		err := os.Mkdir(manDir, 0755)
		if err != nil {
			util.FatalErr("Failed to create directory for man pages.", err)
		}
	}
	err = doc.GenManTree(rootCmd, header, manDir)
	if err != nil {
		util.FatalErr("Failed to generate man pages.", err)
	}
}
