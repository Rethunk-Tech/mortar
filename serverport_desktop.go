//go:build !server

package main

// chooseServerPort has nothing to do in a desktop build, which serves no HTTP port.
func chooseServerPort() {}
