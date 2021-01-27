package main

import (
	"os"
	"path/filepath"

	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	rootCmd := cmd.Root()
	dir := "dist"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fatal.ExitErr(err, "Failed to create dist directory")
	}

	zshCompPath := filepath.Join(dir, "_tb")
	err := rootCmd.GenZshCompletionFile(zshCompPath)
	if err != nil {
		fatal.ExitErr(err, "Failed to create zsh completions")
	}

	bashCompPath := filepath.Join(dir, "tb.bash")
	err = rootCmd.GenBashCompletionFile(bashCompPath)
	if err != nil {
		fatal.ExitErr(err, "Failed to create bash completions")
	}

	manDir := filepath.Join(dir, "man1")
	if err := os.MkdirAll(manDir, 0o755); err != nil {
		fatal.ExitErr(err, "Failed to create directory for man pages.")
	}

	header := &doc.GenManHeader{
		Title:   "tb",
		Section: "1",
	}
	err = doc.GenManTree(rootCmd, header, manDir)
	if err != nil {
		fatal.ExitErr(err, "Failed to generate man pages.")
	}
}
