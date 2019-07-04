package util

import (
	"errors"
	"fmt"
	"os"
)

const npmToken = "NPM_TOKEN"

func NpmLogin() error {
	fmt.Println("Checking private npm repository token...")
	if os.Getenv(npmToken) != "" {
		fmt.Printf("Required env var %s is set\n", npmToken)
		return nil
	}

	fmt.Printf("Required env var %s not set\nChecking ~/.npmrc...\n", npmToken)

	npmrcPath := os.Getenv("HOME") + "/.npmrc"
	if !FileOrDirExists(npmrcPath) {
		fmt.Println("No ~/.npmrc found.")
		fmt.Println("Log in to the touchbistro npm registry with command: 'npm login' and try again.")
		fmt.Println("If this does not work...Create a https://www.npmjs.com/ account called: touchbistro-youremailname, then message DevOps to add you to the @touchbistro account")
		// TODO: We could also let them log in here and continue
		return errors.New("error not logged into npm registry")
	}

	fmt.Println("Looking for token in ~/.npmrc...")

	// figure this thing out
	token := "" // =$(tail -n1 ~/.npmrc | grep -o '//registry.npmjs.org/:_authToken=.*' | cut -f2 -d=)

	if token == "" {
		fmt.Println("could not parse authToken out of ~/.npmrc")
		return errors.New("error no npm token")
	}

	fmt.Println("Found authToken. adding to dotfiles and exporting")
	fmt.Println("...exporting NPM_TOKEN=$token")
	os.Setenv(npmToken, token)

	rcFiles := [2]string{".zshrc", ".bashrc"}

	for _, file := range rcFiles {
		rcPath := os.Getenv("HOME") + "/" + file
		fmt.Printf("...adding export to %s.\n", rcPath)
		AppendLineToFile(rcPath, "export NPM_TOKEN="+token)
		fmt.Println("...done")
	}

	fmt.Println("run 'source ~/.zshrc' or 'source ~/.bashrc'")
	return errors.New("error source rc file") // Why is this a thing?
}
