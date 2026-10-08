//go:build !windows

package launch

import "errors"

// startOwnConsole is never reached off Windows: ownsConsole is always false there.
func startOwnConsole([]string, string, string, []string) (<-chan error, error) {
	return nil, errors.New("a program with its own console is only started on Windows")
}
