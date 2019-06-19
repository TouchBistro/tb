package main

import (
	"fmt"
	"os"

	"github.com/TouchBistro/tb/cmd"
	"github.com/TouchBistro/tb/config"
)

func main() {
	err := config.Init("./config.json")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	err = cmd.RootCmd.Execute()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
