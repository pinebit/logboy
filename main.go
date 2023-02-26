package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pinebit/lognite/app"
)

const defaultConfigPath = "config/config.yaml"

func main() {
	flag.Parse()
	if flag.NArg() > 1 {
		fmt.Println("Usage: lognite [path-to-config-yaml]")
		os.Exit(1)
	}

	configPath := defaultConfigPath
	if flag.NArg() == 1 {
		configPath = flag.Arg(0)
	}

	app := app.NewApp(configPath)
	if err := app.Start(); err != nil {
		log.Fatalf("Application error: %v", err)
	} else {
		log.Println("Application stopped gracefully.")
		os.Exit(0)
	}
}
