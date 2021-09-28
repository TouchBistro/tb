package registry

import (
	"errors"
	"io/fs"

	"github.com/TouchBistro/goutils/color"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/registry"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type validateOptions struct {
	strict bool
}

var validateOpts validateOptions

var validateCmd = &cobra.Command{
	Use:   "validate <path>",
	Args:  cobra.ExactArgs(1),
	Short: "Validates the config files of a registry at the given path.",
	Long: `Validates the config files of a registry at the given path.

Example:
- validates the config files in the current directory
  tb registry validate .`,
	Run: func(cmd *cobra.Command, args []string) {
		registryPath := args[0]
		log.Infof(color.Cyan("Validating registry files at path %s..."), registryPath)

		valid := true
		result := registry.Validate(registryPath, validateOpts.strict, log.StandardLogger())
		if errors.Is(result.AppsErr, fs.ErrNotExist) {
			log.Infof(color.Yellow("No %s file"), registry.AppsFileName)
		} else if result.AppsErr == nil {
			log.Infof(color.Green("✅ %s is valid"), registry.AppsFileName)
		} else {
			log.Infof("❌ %s is invalid\n%v", registry.AppsFileName, result.AppsErr)
			valid = false
		}

		if errors.Is(result.PlaylistsErr, fs.ErrNotExist) {
			log.Infof(color.Yellow("No %s file"), registry.PlaylistsFileName)
		} else if result.PlaylistsErr == nil {
			log.Infof(color.Green("✅ %s is valid"), registry.PlaylistsFileName)
		} else {
			log.Infof("❌ %s is invalid\n%v", registry.PlaylistsFileName, result.PlaylistsErr)
			valid = false
		}

		if errors.Is(result.ServicesErr, fs.ErrNotExist) {
			log.Infof(color.Yellow("No %s file"), registry.ServicesFileName)
		} else if result.ServicesErr == nil {
			log.Infof(color.Green("✅ %s is valid"), registry.ServicesFileName)
		} else {
			log.Infof("❌ %s is invalid\n%v", registry.ServicesFileName, result.ServicesErr)
			valid = false
		}

		if !valid {
			fatal.Exit(color.Red("❌ registry is invalid"))
		}
		log.Info(color.Green("✅ registry is valid"))
	},
}

func init() {
	validateCmd.Flags().BoolVar(&validateOpts.strict, "strict", false, "Strict mode, treat more cases as errors")
	registryCmd.AddCommand(validateCmd)
}
