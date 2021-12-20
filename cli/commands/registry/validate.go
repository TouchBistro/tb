package registry

import (
	"errors"
	"io/fs"

	"github.com/TouchBistro/goutils/color"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/registry"
	"github.com/spf13/cobra"
)

func newValidateCommand(c *cli.Container) *cobra.Command {
	var validateOpts struct {
		strict bool
	}
	validateCmd := &cobra.Command{
		Use:   "validate <path>",
		Args:  cobra.ExactArgs(1),
		Short: "Validates the config files of a registry at the given path.",
		Long: `Validates the config files of a registry at the given path.

	Example:
	- validates the config files in the current directory
	  tb registry validate .`,
		RunE: func(cmd *cobra.Command, args []string) error {
			registryPath := args[0]
			c.Tracker.Infof(color.Cyan("Validating registry files at path %s..."), registryPath)

			valid := true
			result := registry.Validate(registryPath, validateOpts.strict, c.Tracker)
			if errors.Is(result.AppsErr, fs.ErrNotExist) {
				c.Tracker.Infof(color.Yellow("No %s file"), registry.AppsFileName)
			} else if result.AppsErr == nil {
				c.Tracker.Infof(color.Green("✅ %s is valid"), registry.AppsFileName)
			} else {
				c.Tracker.Infof("❌ %s is invalid\n%v", registry.AppsFileName, result.AppsErr)
				valid = false
			}

			if errors.Is(result.PlaylistsErr, fs.ErrNotExist) {
				c.Tracker.Infof(color.Yellow("No %s file"), registry.PlaylistsFileName)
			} else if result.PlaylistsErr == nil {
				c.Tracker.Infof(color.Green("✅ %s is valid"), registry.PlaylistsFileName)
			} else {
				c.Tracker.Infof("❌ %s is invalid\n%v", registry.PlaylistsFileName, result.PlaylistsErr)
				valid = false
			}

			if errors.Is(result.ServicesErr, fs.ErrNotExist) {
				c.Tracker.Infof(color.Yellow("No %s file"), registry.ServicesFileName)
			} else if result.ServicesErr == nil {
				c.Tracker.Infof(color.Green("✅ %s is valid"), registry.ServicesFileName)
			} else {
				c.Tracker.Infof("❌ %s is invalid\n%v", registry.ServicesFileName, result.ServicesErr)
				valid = false
			}

			if !valid {
				return &cli.ExitError{
					Code:    1,
					Message: color.Red("❌ registry is invalid"),
				}
			}
			c.Tracker.Info(color.Green("✅ registry is valid"))
			return nil
		},
	}
	validateCmd.Flags().BoolVar(&validateOpts.strict, "strict", false, "Strict mode, treat more cases as errors")
	return validateCmd
}
