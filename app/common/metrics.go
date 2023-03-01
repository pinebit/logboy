package common

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	PromConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lognite_rpc_alive_connections",
		Help: "The current number of alive RPC connections per chain",
	}, []string{"chainName"})

	PromReConnections = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_rpc_reconnections",
		Help: "The total number of RPC reconnections per chain",
	}, []string{"chainName"})

	PromLogsReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_logs_received",
		Help: "The total number of received logs per chain and contract",
	}, []string{"chainName", "contractName"})

	PromEvents = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_events",
		Help: "The total number of events per chain, contract and event name",
	}, []string{"chainName", "contractName", "eventName"})

	PromEventsMalformed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_events_malformed",
		Help: "The total number of malformed events per chain and contract name",
	}, []string{"chainName", "contractName"})

	PromReorgErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_reorg_errors",
		Help: "The total number of errors due to chain reorgs that may affect data consistency",
	}, []string{"chainName"})

	PromConfiguredEvents = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_configured_events",
		Help: "The total number of events configured per chain and contract",
	}, []string{"chainName", "contractName"})

	PromConfiguredAddresses = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_configured_addresses",
		Help: "The total number of addresses configured per chain and contract",
	}, []string{"chainName", "contractName"})

	PromPostgresErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_postgres_errors",
		Help: "The total number of Postgres errors per table",
	}, []string{"table"})

	PromPostgresInserts = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_postgres_inserts",
		Help: "The total number of Postgres inserts per table",
	}, []string{"table"})

	PromPostgresDrops = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_postgres_drops",
		Help: "The total number of Postgres drops per table",
	}, []string{"table"})

	PromQueueDiscarded = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_queue_discarded",
		Help: "The total number of discarded items per queue",
	}, []string{"queue"})
)
