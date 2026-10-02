//go:build windows

package cli

import "errors"

func freeSpace(string) (uint64, error) {
	return 0, errors.New("free-space check is unavailable")
}
