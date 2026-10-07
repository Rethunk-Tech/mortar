package queue

import (
	"errors"
	"io"
	"runtime"

	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// Failure kinds a failed Item carries, so the GUI picks its wording without parsing Error.
const (
	FailNetwork = "network"
	FailBlocked = "blocked"
	FailAuth    = "auth"
	FailDisk    = "disk"
	FailNoBase  = "nobase"
	FailMalware = "malware"
	FailOther   = "other"
)

// failureKind classifies why a download failed from the error's type.
func failureKind(err error) string {
	var full *store.DiskFullError
	switch {
	case errors.Is(err, nexus.ErrUnauthorized):
		return FailAuth
	case errors.Is(err, nexus.ErrQuarantined):
		return FailBlocked
	case errors.As(err, new(*store.DetectedError)):
		return FailMalware
	// Windows reports an antivirus holding a file Mortar just wrote as access denied; store.RealTimeBlock has
	// already turned its explicit virus errors into a detection.
	case runtime.GOOS == "windows" && usererr.KindOf(err) == usererr.Permission:
		return FailBlocked
	case errors.As(err, &full), usererr.IsDiskFull(err):
		return FailDisk
	case errors.As(err, new(*profile.NoBaseError)):
		return FailNoBase
	case usererr.KindOf(err) == usererr.Network,
		errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		return FailNetwork
	}
	return FailOther
}
