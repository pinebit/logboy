package main

import (
	"github.com/pinebit/obry/app"
)

func main() {
	app := app.NewApp()
	defer app.Close()

	app.Run()
}
