//go:build !windows

package avscan

import "context"

// amsi is the Windows Antimalware Scan Interface; there is none here.
type amsi struct{}

func (amsi) Scan(context.Context, string) (Detection, bool, error) {
	return Detection{}, false, ErrNoScanner
}
