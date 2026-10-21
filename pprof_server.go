//go:build server

package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime/metrics"
	"strings"
	"time"
)

// MORTAR_PPROF=<addr> serves Go's profiling endpoints (/debug/pprof/) in server mode, for the memory harness and for
// profiling a self-test sandbox. Unset, nothing listens; the desktop build never has it. prepareServerMode starts it,
// which only the long-running server reaches, not a CLI call of the same binary.
func servePprof() {
	addr := os.Getenv("MORTAR_PPROF")
	if addr == "" {
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.HandleFunc("/debug/metrics", serveRuntimeMetrics)
	if !strings.Contains(addr, ":") {
		addr = ":" + addr
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      60 * time.Second,
	}
	go func() { log.Printf("pprof: %v", srv.ListenAndServe()) }()
}

// serveRuntimeMetrics writes every scalar runtime/metrics sample as "name value" lines: the memory classes MemStats lacks
// and the GOGC and memory limit actually in effect, which the memory harness reports.
func serveRuntimeMetrics(w http.ResponseWriter, _ *http.Request) {
	descs := metrics.All()
	samples := make([]metrics.Sample, len(descs))
	for i, d := range descs {
		samples[i].Name = d.Name
	}
	metrics.Read(samples)
	for _, s := range samples {
		switch s.Value.Kind() {
		case metrics.KindUint64:
			fmt.Fprintf(w, "%s %d\n", s.Name, s.Value.Uint64())
		case metrics.KindFloat64:
			fmt.Fprintf(w, "%s %g\n", s.Name, s.Value.Float64())
		}
	}
}
