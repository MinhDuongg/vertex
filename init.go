package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/vertex-center/vertex/config"
	"github.com/vertex-center/vertex/pkg/log"
	"github.com/vertex-center/vertex/router"
	"github.com/vertex-center/vertex/services"
	"github.com/vertex-center/vertex/types"
	"github.com/vertex-center/vlog"
)

// version, commit and date will be overridden by goreleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	defer log.Default.Close()

	log.Default.Info("Vertex starting...")

	parseArgs()

	err := setupDependencies()
	if err != nil {
		log.Default.Error(fmt.Errorf("failed to setup dependencies: %v", err))
		return
	}

	err = config.Current.Apply()
	if err != nil {
		log.Default.Error(fmt.Errorf("failed to apply the current configuration: %v", err))
		return
	}

	router := router.NewRouter(types.About{
		Version: version,
		Commit:  commit,
		Date:    date,

		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	})
	defer router.Stop()

	// Logs
	url := fmt.Sprintf("http://%s", config.Current.Host)
	fmt.Printf("\n-- Vertex Client :: %s\n\n", url)
	log.Default.Info("Vertex started",
		vlog.String("url", url),
	)

	router.Start(fmt.Sprintf(":%s", config.Current.Port))
}

func parseArgs() {
	flagVersion := flag.Bool("version", false, "Print vertex version")
	flagV := flag.Bool("v", false, "Print vertex version")
	flagDate := flag.Bool("date", false, "Print the release date")
	flagCommit := flag.Bool("commit", false, "Print the commit hash")

	flagPort := flag.String("port", config.Current.Port, "The Vertex port")
	flagHost := flag.String("host", config.Current.Host, "The Vertex access url")

	flag.Parse()

	if *flagVersion || *flagV {
		fmt.Println(version)
		os.Exit(0)
	}
	if *flagDate {
		fmt.Println(date)
		os.Exit(0)
	}
	if *flagCommit {
		fmt.Println(commit)
		os.Exit(0)
	}
	config.Current.Host = *flagHost
	config.Current.Port = *flagPort
}

func setupDependencies() error {
	dependencies := []types.Dependency{
		services.DependencyClient,
		services.DependencyServices,
		services.DependencyPackages,
	}

	for _, dep := range dependencies {
		err := setupDependency(dep)
		if err != nil {
			log.Default.Error(err)
			os.Exit(1)
		}
	}
	return nil
}

func setupDependency(dep types.Dependency) error {
	err := os.Mkdir(dep.GetPath(), os.ModePerm)
	if os.IsExist(err) {
		// The dependency already exists.
		return nil
	}
	if err != nil {
		return err
	}

	// download
	_, err = dep.CheckForUpdate()
	if err != nil && !errors.Is(err, services.ErrDependencyNotInstalled) {
		return err
	}
	return dep.InstallUpdate()
}
