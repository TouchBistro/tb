package cmd

import (
	"os"
	"strings"

	"time"

	"github.com/TouchBistro/goutils/command"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/spinner"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/deps"
	"github.com/TouchBistro/tb/docker"
	"github.com/TouchBistro/tb/login"
	"github.com/TouchBistro/tb/service"
	"github.com/TouchBistro/tb/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type upOptions struct {
	shouldSkipServicePreRun bool
	shouldSkipGitPull       bool
	shouldSkipDockerPull    bool
	shouldSkipLazydocker    bool
	cliServiceNames         []string
	playlistName            string
}

var upOpts upOptions

func selectServices() []service.Service {
	if len(upOpts.cliServiceNames) > 0 && upOpts.playlistName != "" {
		fatal.Exit("you can only specify one of --playlist or --services.\nTry tb up --help for some examples.")
	}

	var names []string

	// parsing --playlist
	if upOpts.playlistName != "" {
		name := upOpts.playlistName
		if len(name) == 0 {
			fatal.Exit("playlist name cannot be blank. try running tb up --help")
		}
		var err error
		names, err = config.LoadedPlaylists().ServiceNames(name)
		if err != nil {
			fatal.ExitErr(err, "failed resolving service playlist")
		}
		if len(names) == 0 {
			fatal.Exitf("playlist \"%s\" is empty or nonexistent.\ntry running tb list --playlists to see all the available playlists.\n", name)
		}
		// parsing --services
	} else if len(upOpts.cliServiceNames) > 0 {
		names = upOpts.cliServiceNames
	} else {
		fatal.Exit("you must specify either --playlist or --services.\ntry tb up --help for some examples.")
	}

	selectedServices := make([]service.Service, len(names))
	for i, name := range names {
		s, err := config.LoadedServices().Get(name)
		if err != nil {
			fatal.ExitErrf(err, "%s is not a tb service name.\n Try tb list to see all available servies.\n", name)
		}
		selectedServices[i] = s
	}

	if len(selectedServices) == 0 {
		fatal.Exit("you must specify at least one service to run.\nTry tb list --services to see all the available playlists.")
	}

	return selectedServices
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Starts services from a playlist name or as a comma separated list of services",
	Long: `Starts services from a playlist name or as a comma separated list of services.

Examples:
- run the services defined under the "core" key in playlists.yml
	tb up --playlist core

- run only postgres and localstack
	tb up --services postgres,localstack`,
	Args: cobra.NoArgs,
	PreRun: func(cmd *cobra.Command, args []string) {
		err := deps.Resolve(
			deps.Brew,
			deps.Aws,
			deps.Lazydocker,
			deps.Node,
			deps.Yarn,
		)
		if err != nil {
			fatal.ExitErr(err, "could not resolve dependencies")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		selectedServices := selectServices()
		serviceNames := make([]string, len(selectedServices))
		for i, s := range selectedServices {
			serviceNames[i] = s.FullName()
		}
		log.Infof("Running the following services: %s", strings.Join(serviceNames, ", "))

		// We have to clone every possible repo instead of just selected services
		// Because otherwise docker-compose will complaing about missing build paths
		err := config.CloneOrPullRepos(!upOpts.shouldSkipGitPull)
		if err != nil {
			fatal.ExitErr(err, "failed cloning git repos")
		}

		loginStrategies, err := config.LoginStategies()
		if err != nil {
			fatal.ExitErr(err, "Failed to get login strategies")
		}
		if loginStrategies != nil {
			performLoginStrategies(loginStrategies)
		}

		cleanupPrevDocker(selectedServices)
		// check for docker disk usage after cleanup
		full, usage, err := docker.CheckDockerDiskUsage()
		if err != nil {
			fatal.ExitErr(err, "failed checking docker status")
		}
		log.Infof("Current docker disk usage: %.2fGB", float64(usage)/(1024*1024*1024))
		if full {
			if util.Prompt("Your free disk space is running out, would you like to cleanup? y/n >") {
				// Images are all we really care about as far as space cleaning
				s := spinner.New(
					spinner.WithStartMessage("Pruning docker images"),
					spinner.WithStopMessage("Finished pruning docker images"),
					spinner.WithPersistMessages(log.IsLevelEnabled(log.DebugLevel)),
				)
				log.SetOutput(s)
				s.Start()

				pruned, err := docker.PruneImages()
				s.Stop()
				if err != nil {
					fatal.ExitErr(err, "Failed pruning docker images")
				}
				log.SetOutput(os.Stderr)
				log.Infof("Reclaimed %.2fGB", float64(pruned)/(1024*1024*1024))
			} else {
				log.Info("Continuing, but unexpected behavior is possible if docker usage isn't cleaned.")
			}
		}

		if !upOpts.shouldSkipDockerPull {
			pullBaseImages(config.BaseImages())
			pullServiceImages(selectedServices)
		}
		dockerComposeBuild(selectedServices)
		if !upOpts.shouldSkipServicePreRun {
			performPreRun(selectedServices)
		}

		dockerComposeUp(selectedServices)
		if !upOpts.shouldSkipLazydocker {
			w := log.WithField("id", "lazydocker").WriterLevel(log.DebugLevel)
			defer w.Close()
			err := command.New(command.WithStdout(w), command.WithStderr(w)).Exec("lazydocker")
			if err != nil {
				fatal.ExitErr(err, "failed running lazydocker")
			}
		}
		log.Info("🔈 the containers are running in the background. If you want to terminate them, run tb down")
	},
}

func init() {
	upCmd.PersistentFlags().BoolVar(&upOpts.shouldSkipServicePreRun, "no-service-prerun", false, "dont run preRun command for services")
	upCmd.PersistentFlags().BoolVar(&upOpts.shouldSkipGitPull, "no-git-pull", false, "dont update git repositories")
	upCmd.PersistentFlags().BoolVar(&upOpts.shouldSkipDockerPull, "no-remote-pull", false, "dont get new remote images")
	upCmd.PersistentFlags().BoolVar(&upOpts.shouldSkipLazydocker, "no-lazydocker", false, "dont start lazydocker")
	upCmd.PersistentFlags().StringVarP(&upOpts.playlistName, "playlist", "p", "", "the name of a service playlist")
	upCmd.PersistentFlags().StringSliceVarP(&upOpts.cliServiceNames, "services", "s", []string{}, "comma separated list of services to start. eg --services postgres,localstack.")

	rootCmd.AddCommand(upCmd)
}

func performLoginStrategies(loginStrategies []login.LoginStrategy) {
	s := spinner.New(
		spinner.WithStartMessage("Logging into services"),
		spinner.WithStopMessage("Finished logging into services"),
		spinner.WithCount(len(loginStrategies)),
		spinner.WithPersistMessages(log.IsLevelEnabled(log.DebugLevel)),
	)
	log.SetOutput(s)
	defer log.SetOutput(os.Stderr)
	s.Start()

	successCh := make(chan string)
	failedCh := make(chan error)
	for _, s := range loginStrategies {
		log.Debugf("\tLogging into %s", s.Name())
		go func(s login.LoginStrategy) {
			err := s.Login()
			if err != nil {
				failedCh <- err
				return
			}
			successCh <- s.Name()
		}(s)
	}

	for i := 0; i < len(loginStrategies); i++ {
		select {
		case n := <-successCh:
			s.IncWithMessagef("Finished logging into %s", n)
		case err := <-failedCh:
			s.Stop()
			fatal.ExitErr(err, "Failed to log in to services")
		case <-time.After(2 * time.Minute):
			s.Stop()
			fatal.Exit("Timed out while logging in to services")
		}
	}
	s.Stop()
}

func cleanupPrevDocker(services []service.Service) {
	dockerNames := make([]string, len(services))
	for i, s := range services {
		dockerNames[i] = s.DockerName()
	}
	log.Infof("The following services will be restarted if they are running: %s", strings.Join(dockerNames, "\n"))

	s := spinner.New(
		spinner.WithStartMessage("Stopping docker services"),
		spinner.WithStopMessage("Finished removing containers"),
		spinner.WithPersistMessages(log.IsLevelEnabled(log.DebugLevel)),
	)
	log.SetOutput(s)
	defer log.SetOutput(os.Stderr)
	s.Start()

	err := docker.ComposeStop(dockerNames)
	if err != nil {
		s.Stop()
		fatal.ExitErr(err, "failed stopping service containers")
	}
	s.UpdateMessage("Removing old service containers")
	err = docker.ComposeRm(dockerNames)
	s.Stop()
	if err != nil {
		fatal.ExitErr(err, "failed removing old servicec containers")
	}
}

func pullBaseImages(images []string) {
	s := spinner.New(
		spinner.WithStartMessage("Pulling latest base images"),
		spinner.WithStopMessage("Finished pulling latest base images"),
		spinner.WithCount(len(images)),
		spinner.WithPersistMessages(log.IsLevelEnabled(log.DebugLevel)),
	)
	log.SetOutput(s)
	defer log.SetOutput(os.Stderr)
	s.Start()

	successCh := make(chan string)
	failedCh := make(chan error)
	for _, b := range images {
		log.Debugf("\tpulling %s", b)
		go func(b string) {
			err := docker.Pull(b)
			if err != nil {
				failedCh <- err
				return
			}
			successCh <- b
		}(b)
	}

	for i := 0; i < len(images); i++ {
		select {
		case n := <-successCh:
			s.IncWithMessagef("Finished pulling %s", n)
		case err := <-failedCh:
			s.Stop()
			fatal.ExitErr(err, "Failed to pull base images")
		case <-time.After(5 * time.Minute):
			s.Stop()
			fatal.Exit("Timed out while pulling base images")
		}
	}
	s.Stop()
}

func pullServiceImages(services []service.Service) {
	var remoteServices []service.Service
	for _, s := range services {
		if s.UseRemote() {
			remoteServices = append(remoteServices, s)
		}
	}

	s := spinner.New(
		spinner.WithStartMessage("Pulling latest docker images for selected services"),
		spinner.WithStopMessage("Finished pulling latest docker images for selected services"),
		spinner.WithCount(len(remoteServices)),
		spinner.WithPersistMessages(log.IsLevelEnabled(log.DebugLevel)),
	)
	log.SetOutput(s)
	defer log.SetOutput(os.Stderr)
	s.Start()

	successCh := make(chan string)
	failedCh := make(chan error)
	for _, s := range remoteServices {
		log.Debugf("\tpulling image %s", s.ImageURI())
		go func(s service.Service) {
			err := docker.Pull(s.ImageURI())
			if err != nil {
				failedCh <- err
				return
			}
			successCh <- s.FullName()
		}(s)
	}

	for i := 0; i < len(remoteServices); i++ {
		select {
		case n := <-successCh:
			s.IncWithMessagef("Finished pulling image for %s", n)
		case err := <-failedCh:
			s.Stop()
			fatal.ExitErr(err, "Failed to pull docker images")
		case <-time.After(5 * time.Minute):
			s.Stop()
			fatal.Exit("Timed out while pulling docker images")
		}
	}
	s.Stop()
}

func dockerComposeBuild(services []service.Service) {
	log.Debugf("Checking for services that require images built")
	var names []string
	for _, s := range services {
		if !s.UseRemote() {
			names = append(names, s.DockerName())
		}
	}
	if len(names) == 0 {
		log.Debugf("No services to build")
		return
	}

	s := spinner.New(
		spinner.WithStartMessage("Building images for necessary services"),
		spinner.WithStopMessage("Finished building images"),
		spinner.WithPersistMessages(log.IsLevelEnabled(log.DebugLevel)),
	)
	log.SetOutput(s)
	defer log.SetOutput(os.Stderr)
	s.Start()

	err := docker.ComposeBuild(names)
	s.Stop()
	if err != nil {
		fatal.ExitErr(err, "Failed to build docker images for services")
	}
}

func performPreRun(services []service.Service) {
	var svcs []service.Service
	for _, s := range services {
		if s.PreRun != "" {
			svcs = append(svcs, s)
		}
	}

	s := spinner.New(
		spinner.WithStartMessage("Performing preRun step for services, this may take a long time"),
		spinner.WithStopMessage("Finished performing all preRun steps"),
		spinner.WithCount(len(svcs)),
		spinner.WithPersistMessages(log.IsLevelEnabled(log.DebugLevel)),
	)
	log.SetOutput(s)
	defer log.SetOutput(os.Stderr)
	s.Start()

	successCh := make(chan string)
	failedCh := make(chan error)
	for _, s := range svcs {
		log.Debugf("\trunning preRun command %s for %s", s.PreRun, s.FullName())
		go func(s service.Service) {
			err := docker.ComposeRun(s.DockerName(), s.PreRun)
			if err != nil {
				failedCh <- err
				return
			}
			successCh <- s.FullName()
		}(s)
		// We need to wait a bit in between launching goroutines or else they all create seperated docker-compose environments
		// Any ideas better than a sleep hack are appreciated
		time.Sleep(time.Second)
	}

	for i := 0; i < len(svcs); i++ {
		select {
		case n := <-successCh:
			s.IncWithMessagef("Finished running preRun command for %s", n)
		case err := <-failedCh:
			s.Stop()
			fatal.ExitErr(err, "Failed running preRun commands")
		case <-time.After(5 * time.Minute):
			s.Stop()
			fatal.Exit("Timed out while running preRun commands")
		}
	}
	s.Stop()
}

func dockerComposeUp(services []service.Service) {
	log.Debugf("starting docker-compose up in detached mode")
	dockerNames := make([]string, len(services))
	for i, s := range services {
		dockerNames[i] = s.DockerName()
	}

	err := docker.ComposeUp(dockerNames)
	if err != nil {
		fatal.ExitErr(err, "Could not run docker-compose up")
	}
	log.Debugf("Finished starting docker-compose up in detached mode")
}
