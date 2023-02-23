package app

import (
	"os"
	"strings"
)

func UnwrapConfigEnvVar(value string) string {
	if strings.HasPrefix(value, "$") {
		return os.Getenv(strings.TrimPrefix(value, "$"))
	}
	return value
}
