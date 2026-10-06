//go:build !server

package main

// prepareServerMode has nothing to check in a desktop build, which has a single-instance lock and serves no HTTP port.
func prepareServerMode() error { return nil }

// forwardLaunch is the server build's hand-off; a desktop build's single-instance lock carries a second launch.
func forwardLaunch(string, []string) bool { return false }
