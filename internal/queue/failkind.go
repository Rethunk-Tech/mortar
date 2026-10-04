package queue

import (
	"errors"
	"io"

	"github.com/Rethunk-AI/mortar/internal/nexus"
	"github.com/Rethunk-AI/mortar/internal/store"
	"github.com/Rethunk-AI/mortar/internal/usererr"
)

// Failure kinds a failed Item carries, so the GUI picks its wording without parsing Error.
const (
	FailNetwork = "network"
	FailBlocked = "blocked"
	FailAuth    = "auth"
	FailDisk    = "disk"
	FailOther   = "other"
)

// failureKind classifies why a download failed from the error's type.
func failureKind(err error) string {
	var full *store.DiskFullError
	switch {
	case errors.Is(err, nexus.ErrUnauthorized):
		return FailAuth
	case errors.Is(err, nexus.ErrQuarantined), platformBlocked(err):
		return FailBlocked
	case errors.As(err, &full), usererr.IsDiskFull(err):
		return FailDisk
	case usererr.KindOf(err) == usererr.Network,
		errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		return FailNetwork
	}
	return FailOther
}
