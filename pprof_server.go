//go:build server

package main

import (
	"log"
	"net/http"
	"net/http/pprof"
	"os"
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
	go func() { log.Printf("pprof: %v", http.ListenAndServe(addr, mux)) }()
}
