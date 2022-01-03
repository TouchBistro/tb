package commands

import (
	"fmt"
	"os"

	"github.com/TouchBistro/tb/cli"
	"github.com/spf13/cobra"
)

func newCompletionsCommand() *cobra.Command {
	return &cobra.Command{
		Use:       "completions <shell>",
		Args:      cobra.ExactValidArgs(1),
		ValidArgs: []string{"bash", "zsh"},
		Short:     "Generate shell completions",
		Long: `Generates a shell completion script and outputs it to standard output.

Supported shells are: bash, zsh.

For example to generate and use bash completions:

	shed completions bash > /usr/local/etc/bash_completion.d/shed.bash
	source /usr/local/etc/bash_completion.d/shed.bash`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Provide an empty function to override the one in the root command.
			// We want to skip all pre-run steps for this command since none of that is relevant.
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := args[0]
			var err error
			switch shell {
			case "bash":
				err = cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				err = cmd.Root().GenZshCompletion(os.Stdout)
			default:
				return &cli.ExitError{
					Message: fmt.Sprintf(
						"Unsupported shell %q. Run 'tb completions --help' to see supported shells.",
						shell,
					),
				}
			}
			if err != nil {
				return fmt.Errorf("failed to generate %s completions: %w", shell, err)
			}
			return nil
		},
	}
}
