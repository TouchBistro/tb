package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/engine"
	"github.com/spf13/cobra"
)

type upOptions struct {
	skipServicePreRun bool
	skipGitPull       bool
	skipDockerPull    bool
	skipLazydocker    bool
	playlistName      string
	serviceNames      []string
	serviceTags       []string
}

func parseTag(tag string, serviceNames []string) ([]string, error) {
	parts := strings.Split(tag, ":")
	// when running one service with one part ex: tb up some-service -t some-tag
	if len(serviceNames) == 1 && len(parts) == 1 {
		return []string{serviceNames[0], parts[0]}, nil

	}

	// split the service tag into service name and tag, ensuring exactly two string values
	// ex: tb up some-service -t some-service:some-tag
	if len(parts) != 2 {
		return []string{}, fmt.Errorf("invalid service tag format '%s'; expected format 'service:tag'", tag)
	}

	if parts[0] == "" || parts[1] == "" {
		return []string{}, fmt.Errorf("invalid service tag '%s'; service name and tag must not be empty", tag)
	}

	return parts, nil
}

func newUpCommand(c *cli.Container) *cobra.Command {
	var opts upOptions
	upCmd := &cobra.Command{
		Use: "up [services...]",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && opts.playlistName == "" && len(opts.serviceNames) == 0 {
				return fmt.Errorf("service names or playlist name is required")
			}
			if len(args) > 0 && opts.playlistName != "" {
				return fmt.Errorf("cannot specify service names as args when --playlist or -p is used")
			}
			// These are deprecated and will be removed but we need to check for it for now for backwards compatibility
			if len(args) > 0 && len(opts.serviceNames) > 0 {
				return fmt.Errorf("cannot specify service names as args when --services or -s is used")
			}
			if len(opts.serviceNames) > 0 && opts.playlistName != "" {
				return fmt.Errorf("cannot specify both --playist,-p and --services,-s")
			}
			return nil
		},
		Short: "Start services or playlists",
		Long: `Starts one or more services. The following actions will be performed before starting services:

- Stop and remove any services that are already running.
- Pull base images and service images.
- Build any services with mode build.
- Run pre-run steps for services.

Services can be specified in one of two ways. First, the names of the services can be specified directly as args.
Second, the --playlist,-p flag can be used to provide a playlist name in order to start all the services in the playlist.
If a playlist is provided no args can be provided, that is, mixing a playlist and service names is not allowed.

Examples:

Run the services defined in the 'core' playlist in a registry:

	tb up --playlist core

Run the postgres and localstack services directly:

	tb up postgres localstack`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Hack to support either args or --services flag for backwards compatibility.
			// The flag will eventually be removed so we won't have to do this
			// and will be able to just pass args to Engine.Up.
			serviceNames := args
			if len(serviceNames) == 0 {
				serviceNames = opts.serviceNames
			}

			serviceTags := make(map[string]string)
			for _, serviceTag := range opts.serviceTags {
				parts, err := parseTag(serviceTag, serviceNames)
				if err != nil {
					return &fatal.Error{
						Msg: fmt.Sprintf("Failed to parse service tag %s", serviceTag),
						Err: err,
					}
				}
				serviceTags[parts[0]] = parts[1]
			}
			err := c.Engine.Up(c.Ctx, engine.UpOptions{
				ServiceNames:   serviceNames,
				PlaylistName:   opts.playlistName,
				SkipPreRun:     opts.skipServicePreRun,
				SkipDockerPull: opts.skipDockerPull,
				SkipGitPull:    opts.skipGitPull,
				OfflineMode:    c.OfflineMode,
				ServiceTags:    serviceTags,
			})
			if err != nil {
				return &fatal.Error{
					Msg: "Failed to start services",
					Err: err,
				}
			}
			c.Tracker.Info("✔ Started services")

			if !opts.skipLazydocker {
				// lazydocker opt in, if it exists it will be launched, otherwise this step will be skipped
				const lazydocker = "lazydocker"
				if path, err := exec.LookPath(lazydocker); err == nil {
					c.Tracker.Debug("Running lazydocker")
					// Lazydocker doesn't write to stdout or stderr since everything is displaed in the terminal GUI
					if err := syscall.Exec(path, []string{path}, os.Environ()); err != nil {
						return &fatal.Error{
							Msg: "Failed running lazydocker",
							Err: err,
						}
					}
				} else {
					// Skip, but inform users about installing it
					c.Tracker.Warnf("lazydocker is not installed. Consider installing it: https://github.com/jesseduffield/lazydocker#installation")
				}
			}
			c.Tracker.Info("🔈 the containers are running in the background. If you want to terminate them, run tb down")
			return nil
		},
	}

	flags := upCmd.Flags()
	flags.BoolVar(&opts.skipServicePreRun, "no-service-prerun", false, "Don't run preRun command for services")
	flags.BoolVar(&opts.skipGitPull, "no-git-pull", false, "Don't update git repositories")
	flags.BoolVar(&opts.skipDockerPull, "no-remote-pull", false, "Don't get new remote images")
	flags.BoolVar(&opts.skipLazydocker, "no-lazydocker", false, "Don't start lazydocker")
	flags.StringVarP(&opts.playlistName, "playlist", "p", "", "The name of a playlist")
	flags.StringSliceVarP(&opts.serviceTags, "image-tag", "t", []string{}, "Comma separated list of service:image-tag to run")
	flags.StringSliceVarP(&opts.serviceNames, "services", "s", []string{}, "Comma separated list of services to start. eg --services postgres,localstack.")
	err := flags.MarkDeprecated("services", "and will be removed, pass service names as arguments instead")
	if err != nil {
		// MarkDeprecated only errors if the flag name is wrong or the message isn't set
		// which is a programming error, so we wanna blow up
		panic(err)
	}
	return upCmd
}
