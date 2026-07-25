package types

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type Metrics struct {
	BlocksMined  uint64
	BlocksFailed uint64
	TxProcessed  uint64
}

var GlobalMetrics Metrics

func PrometheusHandler(w http.ResponseWriter, r *http.Request) {
	mined := atomic.LoadUint64(&GlobalMetrics.BlocksMined)
	failed := atomic.LoadUint64(&GlobalMetrics.BlocksFailed)
	txs := atomic.LoadUint64(&GlobalMetrics.TxProcessed)

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# HELP sovereign_blocks_mined_total Total blocks successfully mined.\n")
	fmt.Fprintf(w, "# TYPE sovereign_blocks_mined_total counter\n")
	fmt.Fprintf(w, "sovereign_blocks_mined_total %d\n\n", mined)

	fmt.Fprintf(w, "# HELP sovereign_blocks_failed_total Total blocks failed mining.\n")
	fmt.Fprintf(w, "# TYPE sovereign_blocks_failed_total counter\n")
	fmt.Fprintf(w, "sovereign_blocks_failed_total %d\n\n", failed)

	fmt.Fprintf(w, "# HELP sovereign_txs_processed_total Total transactions processed.\n")
	fmt.Fprintf(w, "# TYPE sovereign_txs_processed_total counter\n")
	fmt.Fprintf(w, "sovereign_txs_processed_total %d\n", txs)
}
