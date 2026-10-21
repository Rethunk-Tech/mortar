//go:build !server

package main

// serverBuild is false in the desktop build, whose window can hand an address to the system's browser.
const serverBuild = false

// prepareServerMode has nothing to check in a desktop build, which has a single-instance lock and serves no HTTP port.
func prepareServerMode() error { return nil }

// forwardLaunch is the server build's hand-off; a desktop build's single-instance lock carries a second launch.
func forwardLaunch(string, []string) bool { return false }
