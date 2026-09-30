//go:build !linux && !windows

package nxm

import "errors"

// System is the system's registration of the nxm scheme.
type System struct{}

// New returns the handler for this system, which has none.
func New(string) (System, error) { return System{}, nil }

var errUnsupported = errors.New("handling nxm links is not supported on this system")

func (System) Owner() (Owner, error) { return Owner{}, errUnsupported }
func (System) Register() error       { return errUnsupported }
func (System) Restore(string) error  { return errUnsupported }
func (System) RegisterLinks() error  { return nil }
func (System) Refresh() error        { return nil }
