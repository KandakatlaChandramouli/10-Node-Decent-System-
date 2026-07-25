package runtime

import (
	"context"
	"net/http"
	"net/http/pprof"
	"sync"
	"sync/atomic"
	"time"
)

type RuntimeObservability struct {
	mu            sync.RWMutex
	server        *http.Server
	txProcessed   atomic.Uint64
	latencySumMs  atomic.Uint64
	activeWorkers atomic.Int64
}

func NewRuntimeObservability() *RuntimeObservability {
	return &RuntimeObservability{}
}

func (ro *RuntimeObservability) StartPprofServer(addr string) {
	ro.mu.Lock()
	defer ro.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	ro.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		_ = ro.server.ListenAndServe()
	}()
}

func (ro *RuntimeObservability) RecordExecution(duration time.Duration) {
	ro.txProcessed.Add(1)
	ro.latencySumMs.Add(uint64(duration.Milliseconds()))
}

func (ro *RuntimeObservability) GetMetrics() (uint64, uint64) {
	count := ro.txProcessed.Load()
	totalLat := ro.latencySumMs.Load()
	var avgLat uint64
	if count > 0 {
		avgLat = totalLat / count
	}
	return count, avgLat
}

func (ro *RuntimeObservability) Stop(ctx context.Context) error {
	ro.mu.Lock()
	defer ro.mu.Unlock()
	if ro.server != nil {
		return ro.server.Shutdown(ctx)
	}
	return nil
}
