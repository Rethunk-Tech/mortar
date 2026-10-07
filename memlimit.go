package main

import "runtime/debug"

// goMemoryLimit is the Go runtime's soft heap target. The runtime collects harder as the heap nears it and never
// refuses an allocation, so a large install or check may pass it. It is the lever on the Go side's peak: a Problems
// check on 811 mods keeps about 245 MiB live, and with the collector's default headroom the heap reached 450 MiB; at
// 320 MiB the process peaked 105 MiB lower for the same wall time, which keeps the Go process and the WebView under the 1 GB they share.
const goMemoryLimit = 320 << 20

// applyMemoryLimit sets the soft limit unless GOMEMLIMIT already names one, and reports whether it did.
func applyMemoryLimit(getenv func(string) string) bool {
	if getenv("GOMEMLIMIT") != "" {
		return false
	}
	debug.SetMemoryLimit(goMemoryLimit)
	return true
}
