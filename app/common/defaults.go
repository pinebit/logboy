package common

import "time"

const (
	DefaultServerPort            uint16        = 8080
	DefaultQueueCapacity         int           = 32
	DefaultPostgresRetention     time.Duration = 24 * time.Hour
	DefaultPostgresPruneInterval time.Duration = time.Hour
)
