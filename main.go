package main

import (
	"os"

	"github.com/pinebit/obry/app"
)

const defaultConfigPath = "config/config.json"

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func getConfigPath() string {
	if len(os.Args) >= 2 {
		candidate := os.Args[1]
		if fileExists(candidate) {
			return candidate
		}
	}
	return defaultConfigPath
}

func main() {
	configPath := getConfigPath()
	app := app.NewApp(configPath)
	defer app.Close()

	app.Run()
}
