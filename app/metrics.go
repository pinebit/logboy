package app

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	promConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lognite_rpc_alive_connections",
		Help: "The current number of alive RPC connections per chain",
	}, []string{"chainName"})

	promReConnections = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_rpc_reconnections",
		Help: "The total number of RPC reconnections per chain",
	}, []string{"chainName"})

	promLogsReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_logs_received",
		Help: "The total number of received logs per chain",
	}, []string{"chainName"})

	promEvents = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_events",
		Help: "The total number of events per contract, address and event name",
	}, []string{"chainName", "contractName", "contractAddress", "eventName"})

	promConfiguredEvents = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_configured_events",
		Help: "The total number of events configured per chain and contract",
	}, []string{"chainName", "contractName"})

	promConfiguredAddresses = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_configured_addresses",
		Help: "The total number of addresses configured per chain and contract",
	}, []string{"chainName", "contractName"})

	promDBErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_db_errors",
		Help: "The total number of DB errors per table",
	}, []string{"table"})

	promDBInserts = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_db_inserts",
		Help: "The total number of DB inserts per table",
	}, []string{"table"})

	promDBDrops = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_db_drops",
		Help: "The total number of DB drops per table",
	}, []string{"table"})
)
