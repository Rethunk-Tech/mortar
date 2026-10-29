//go:build unix && !linux && !darwin

package launch

import "errors"

// WaitPID is unsupported here: x/sys has no kqueue exit status for the BSDs, and Mortar ships no build for them.
func WaitPID(int) (Exit, error) {
	return Exit{}, errors.New("waiting on another process is not supported on this system")
}
