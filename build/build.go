package main

import (
	"fmt"
	"log"
	"os"

	"github.com/TouchBistro/tb/cmd"
	"github.com/TouchBistro/tb/util"
	"github.com/spf13/cobra/doc"
)

func main() {
	rootCmd := cmd.Root()
	dir := "dist"
	if !util.FileOrDirExists(dir) {
		os.Mkdir(dir, 0755)
	}

	zshCompPath := fmt.Sprintf("%s/_tb", dir)
	err := rootCmd.GenZshCompletionFile(zshCompPath)
	if err != nil {
		log.Fatal(err)
	}

	bashCompPath := fmt.Sprintf("%s/tb.bash", dir)
	err = rootCmd.GenBashCompletionFile(bashCompPath)
	if err != nil {
		log.Fatal(err)
	}

	header := &doc.GenManHeader{
		Title:   "tb",
		Section: "1",
	}

	manDir := dir + "/man1"
	if !util.FileOrDirExists(manDir) {
		os.Mkdir(manDir, 0755)
	}
	err = doc.GenManTree(rootCmd, header, manDir)
	if err != nil {
		log.Fatal(err)
	}
}
