package cmd

import (
  "fmt"
  "sort"

  "github.com/TouchBistro/goutils/fatal"
  "github.com/TouchBistro/tb/config"
  "github.com/TouchBistro/tb/service"
  "github.com/spf13/cobra"
)

var docsCmd = &cobra.Command{
  Use:     "docs",
  Args:    cobra.noArgs,
  Short:   "Opens link to API docs for a given service",
  Run: func(cmd *cobra.Command, args []string) {
    serviceName := args[0]
    service, err := config.LoadedServices.Get(serviceName)
    if err != nil {
      fatal.ExitErrf(err, "%s is not a valid service.\nTry running `tb list` to see available services\n", serviceName)
    }

    fmt.Println("Opening API docs for: %s", serviceName)
    openDocs(&service)
  },
}

func openDocs(service *service) {
  fmt.Println("hals")
}

func init() {
  rootCmd.AddCommand(docsCmd)
  upCmd.PersistentFlags().StringSliceVarP("service", "s", string, "service to lookup API docs for")
}
