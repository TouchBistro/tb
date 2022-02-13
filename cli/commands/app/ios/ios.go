package ios

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/engine"
	"github.com/spf13/cobra"
)

func NewiOSCommand(c *cli.Container) *cobra.Command {
	iosCmd := &cobra.Command{
		Use:   "ios",
		Short: "Run and manage iOS apps",
		Long:  `tb app ios allows running and managing iOS apps.`,
	}
	iosCmd.AddCommand(newLogsCommand(c), newRunCommand(c))
	return iosCmd
}

func resolveDeviceName(c *cli.Container, appName, iosVersion, deviceName string) (resolvediOSVersion, resolvedDeviceName string, err error) {
	if deviceName != "" {
		// deviceName was provided so use that, even if iosVersion is empty it will be resolved later.
		return iosVersion, deviceName, nil
	}
	// Prompt the user to select a device
	var deviceNames []string
	deviceNames, resolvediOSVersion, err = c.Engine.AppiOSListDevices(c.Ctx, engine.AppiOSListDevicesOptions{
		AppName:    appName,
		IOSVersion: iosVersion,
	})
	if err != nil {
		return "", "", &fatal.Error{
			Msg: "Failed to get list of iOS devices",
			Err: err,
		}
	}

	prompt := &survey.Select{
		Message: "Select iOS simulator device to use:",
		Options: deviceNames,
	}
	var selected string
	if err := survey.AskOne(prompt, &selected); err != nil {
		return "", "", &fatal.Error{
			Msg: "Failed to prompt for iOS device",
			Err: err,
		}
	}
	return resolvediOSVersion, selected, nil
}
