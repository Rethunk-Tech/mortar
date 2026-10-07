package main

import (
	"runtime/debug"
	"testing"
)

func TestMemoryLimitYieldsToGOMEMLIMIT(t *testing.T) {
	before := debug.SetMemoryLimit(-1)
	t.Cleanup(func() { debug.SetMemoryLimit(before) })
	if applyMemoryLimit(func(string) string { return "256MiB" }) || debug.SetMemoryLimit(-1) != before {
		t.Fatal("changed the limit although GOMEMLIMIT is set")
	}
	if !applyMemoryLimit(func(string) string { return "" }) || debug.SetMemoryLimit(-1) != goMemoryLimit {
		t.Fatalf("limit = %d", debug.SetMemoryLimit(-1))
	}
}
