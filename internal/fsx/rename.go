package fsx

import (
	"os"
	"time"
)

const renameRetryWindow = 2 * time.Second

// Rename is os.Rename that rides out the short Windows window in which an antivirus scanner or the search indexer
// holds a freshly written file open. Elsewhere it is os.Rename.
func Rename(from, to string) error {
	return retry(func() error { return os.Rename(from, to) }, transientRename, renameRetryWindow)
}

// retry runs op until it succeeds, fails with a non-transient error, or window has passed, backing off between tries.
func retry(op func() error, transient func(error) bool, window time.Duration) error {
	start := time.Now()
	for delay := time.Millisecond; ; delay *= 2 {
		err := op()
		if err == nil || !transient(err) || time.Since(start)+delay > window {
			return err
		}
		time.Sleep(delay)
	}
}

// RemoveAll is os.RemoveAll with the same antivirus/indexer retry as Rename; a repeat pass removes whatever the
// first one could not.
func RemoveAll(path string) error {
	return retry(func() error { return os.RemoveAll(path) }, transientRename, renameRetryWindow)
}
