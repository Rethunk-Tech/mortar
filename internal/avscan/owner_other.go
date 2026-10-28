//go:build !unix

package avscan

func trustedSocketOwner(string) bool { return true }
