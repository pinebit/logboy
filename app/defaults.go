package app

import "time"

const (
	defaultServerPort            uint16        = 8080
	defaultOutputBuffer          int           = 32
	defaultPostgresRetention     time.Duration = 24 * time.Hour
	defaultPostgresPruneInterval time.Duration = time.Hour
)
