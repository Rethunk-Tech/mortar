//go:build windows

package datadir

import (
	"os"

	"golang.org/x/sys/windows"
)

// linkedKey returns the file-index key of a file with more than one hard link. os.FileInfo does not carry the
// link count on Windows, so the file is opened for attributes only.
func linkedKey(path string, _ os.FileInfo) (fileKey, bool) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return fileKey{}, false
	}
	h, err := windows.CreateFile(p, windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return fileKey{}, false
	}
	defer func() { _ = windows.CloseHandle(h) }()
	var fi windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(h, &fi); err != nil || fi.NumberOfLinks < 2 {
		return fileKey{}, false
	}
	return fileKey{uint64(fi.VolumeSerialNumber), uint64(fi.FileIndexHigh)<<32 | uint64(fi.FileIndexLow)}, true
}
