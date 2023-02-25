package main

import (
	"flag"
	"fmt"
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
	defer app.Close()

	app.Run()
}
