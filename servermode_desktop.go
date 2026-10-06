//go:build !server

package main

// prepareServerMode has nothing to check in a desktop build, which has a single-instance lock and serves no HTTP port.
func prepareServerMode() error { return nil }
