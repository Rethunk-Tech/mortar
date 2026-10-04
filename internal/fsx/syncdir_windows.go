//go:build windows

package fsx

// SyncDir does nothing: NTFS journals directory metadata and a directory handle cannot be flushed.
func SyncDir(string) error { return nil }
