package main

import (
	"os"
	"runtime/debug"
)

// goMemoryLimit is the Go runtime's soft heap target. The runtime collects harder as the heap nears it and never
// refuses an allocation, so a large install or check may pass it; it keeps the Go side near half the 1 GB budget that
// the Go process and the WebView share.
const goMemoryLimit = 512 << 20

// applyMemoryLimit sets the soft limit unless GOMEMLIMIT already names one, and reports whether it did.
func applyMemoryLimit(getenv func(string) string) bool {
	if getenv("GOMEMLIMIT") != "" {
		return false
	}
	debug.SetMemoryLimit(goMemoryLimit)
	return true
}

func init() { applyMemoryLimit(os.Getenv) }
