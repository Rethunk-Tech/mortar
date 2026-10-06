//go:build server

package game

import "os"

// launchWrapper is the program a server build starts every direct game launch through, named by the self-test sandbox.
func launchWrapper() string { return os.Getenv("MORTAR_LAUNCH_WRAPPER") }
