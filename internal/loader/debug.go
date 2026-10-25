package loader

import "context"

// DebugAddress is where a launch started for debugging listens for a Mono debugger (VS Code's Unity debugger, Rider).
const DebugAddress = "127.0.0.1:10000"

type debugKey struct{}

// WithDebugger marks a launch as started for debugging: a driver that can, opens the game's Mono debugger on
// DebugAddress, and one that cannot refuses the launch.
func WithDebugger(ctx context.Context) context.Context {
	return context.WithValue(ctx, debugKey{}, true)
}

// Debugger reports whether the launch was started for debugging.
func Debugger(ctx context.Context) bool {
	on, _ := ctx.Value(debugKey{}).(bool)
	return on
}
