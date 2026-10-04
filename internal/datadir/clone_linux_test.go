//go:build linux

package datadir

import (
	"testing"

	"golang.org/x/sys/unix"
)

func TestLinkFallbackOnFATRefusal(t *testing.T) {
	if !isLinkFallback(unix.EPERM) {
		t.Fatal("EPERM from a link on FAT/exFAT must fall through to a copy")
	}
}
