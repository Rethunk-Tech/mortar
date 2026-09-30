//go:build !windows

package updatesvc

func onApplied(string) func(string) { return nil }
