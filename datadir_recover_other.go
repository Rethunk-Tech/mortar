//go:build !windows

package main

// There is no pre-window dialog off Windows; the error reaches the terminal or journal and the next start can fix
// data-location by hand.
func askMissingLocation(string) locationChoice { return choiceNoDialog }

func showStartupError(string) {}

func pickDataFolder() string { return "" }
