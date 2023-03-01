package common

import "time"

const (
	DefaultServerPort            uint16        = 8080
	DefaultPosgresQueueCapacity  int           = 256
	DefaultPostgresRetention     time.Duration = 24 * time.Hour
	DefaultPostgresPruneInterval time.Duration = 10 * time.Minute
	DefaultConfirmations         uint          = 3
	DefaultBackfillInterval      time.Duration = 100 * time.Millisecond
)
