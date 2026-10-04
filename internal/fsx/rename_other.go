//go:build !windows

package fsx

func transientRename(error) bool { return false }
