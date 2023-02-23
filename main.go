package main

import (
	"github.com/pinebit/lognite/app"
)

const defaultConfigPath = "config/config.yaml"

func main() {
	app := app.NewApp(defaultConfigPath)
	defer app.Close()

	app.Run()
}
