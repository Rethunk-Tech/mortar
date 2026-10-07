package store

import (
	"errors"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/Rethunk-Tech/mortar/internal/avscan"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// RealTimeScanner names the antivirus that blocks a file as Mortar writes or reads it, as opposed to one Mortar
// asked to scan.
const RealTimeScanner = "Windows real-time protection"

// ERROR_VIRUS_INFECTED and ERROR_VIRUS_DELETED: Windows refuses a file operation because the antivirus removed or
// blocked the file.
const (
	errVirusInfected = syscall.Errno(225)
	errVirusDeleted  = syscall.Errno(226)
)

func isVirusErrno(err error) bool {
	return errors.Is(err, errVirusInfected) || errors.Is(err, errVirusDeleted)
}

// RealTimeBlock reports err (of the item stored under key, when the caller knows it) as a malware refusal when the antivirus's real-time protection caused it: a virus errno
// on a write, open or read, or a file that vanished from one of the staging folders Mortar writes into. The scanner
// never says what it found, so the detection has no name, and the file is gone, so there is nothing to install
// anyway. Any other error, including a detection already reported, comes back as it was.
func RealTimeBlock(game, key string, err error, staging ...string) error {
	if err == nil || usererr.KindOf(err) == usererr.Malware {
		return err
	}
	if (runtime.GOOS != "windows" || !isVirusErrno(err)) && !vanishedFrom(err, staging) {
		return err
	}
	hit := avscan.Detection{Scanner: RealTimeScanner}
	return usererr.Wrap(usererr.Malware, &DetectedError{Game: game, Key: key, Removed: true, Detection: hit})
}

func vanishedFrom(err error, staging []string) bool {
	var pe *fs.PathError
	if !errors.Is(err, fs.ErrNotExist) || !errors.As(err, &pe) {
		return false
	}
	for _, dir := range staging {
		if rel, relErr := filepath.Rel(dir, pe.Path); relErr == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
