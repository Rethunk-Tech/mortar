//go:build windows

package doctor

import "errors"

func freeSpace(string) (uint64, error) {
	return 0, errors.New("free-space check is unavailable")
}
