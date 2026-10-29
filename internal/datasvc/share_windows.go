//go:build windows

package datasvc

import (
	"os"

	"golang.org/x/sys/windows"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// fileAlloc identifies the file by volume serial and file index, so the names of a hard link count once. Windows
// reports no cheap allocated size, so the logical size stands in for it.
func fileAlloc(path string, info os.FileInfo) (dev, ino uint64, allocated int64, ok bool) {
	p, err := windows.UTF16PtrFromString(fsx.ExtendedPath(path))
	if err != nil {
		return 0, 0, 0, false
	}
	h, err := windows.CreateFile(p, windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return 0, 0, 0, false
	}
	defer func() { _ = windows.CloseHandle(h) }()
	var fi windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(h, &fi); err != nil {
		return 0, 0, 0, false
	}
	return uint64(fi.VolumeSerialNumber), uint64(fi.FileIndexHigh)<<32 | uint64(fi.FileIndexLow), info.Size(), true
}
