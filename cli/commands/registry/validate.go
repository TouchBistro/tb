package registry

import (
	"errors"
	"io/fs"

	"github.com/TouchBistro/goutils/color"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/registry"
	"github.com/spf13/cobra"
)

type validateOptions struct {
	strict bool
}

func newValidateCommand(c *cli.Container) *cobra.Command {
	var opts validateOptions
	validateCmd := &cobra.Command{
		Use:   "validate <path>",
		Args:  cli.ExpectSingleArg("registry path"),
		Short: "Validate the config files in a registry",
		Long: `Validates the config files in a registry at the given path.

Examples:

Validate the config files in the current directory:

	tb registry validate .`,
		RunE: func(cmd *cobra.Command, args []string) error {
			registryPath := args[0]
			c.Tracker.Infof(color.Cyan("Validating registry files at path %s..."), registryPath)

			valid := true
			result := registry.Validate(registryPath, registry.ValidateOptions{
				Strict: opts.strict,
				Logger: c.Tracker,
			})
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
					Message: color.Red("❌ registry is invalid"),
				}
			}
			c.Tracker.Info(color.Green("✅ registry is valid"))
			return nil
		},
	}

	flags := validateCmd.Flags()
	flags.BoolVar(&opts.strict, "strict", false, "Strict mode, treat more cases as errors")
	return validateCmd
}
