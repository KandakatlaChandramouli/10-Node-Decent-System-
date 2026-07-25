package types

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var (
	BlocksMinedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sovereign_blocks_mined_total",
			Help: "Total blocks successfully mined.",
		},
	)
	BlocksFailedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sovereign_blocks_failed_total",
			Help: "Total blocks failed mining.",
		},
	)
	TxProcessedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sovereign_txs_processed_total",
			Help: "Total transactions processed.",
		},
	)
)

func init() {
	prometheus.MustRegister(BlocksMinedTotal)
	prometheus.MustRegister(BlocksFailedTotal)
	prometheus.MustRegister(TxProcessedTotal)
}

func PrometheusHandler() http.Handler {
	return promhttp.Handler()
}
