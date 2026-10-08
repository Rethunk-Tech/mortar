//go:build windows

package datadir

import (
	"encoding/binary"
	"os"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"golang.org/x/sys/windows"
)

// fileIDInfoSize is FILE_ID_INFO: a 64-bit volume serial and the 128-bit file id that stays unique on ReFS, where the
// 64-bit file index may not.
const fileIDInfoSize = 24

// linkedKey returns the file-id key of a file with more than one hard link. os.FileInfo does not carry the
// link count on Windows, so the file is opened for attributes only.
func linkedKey(path string, _ os.FileInfo) (fileKey, bool) {
	p, err := windows.UTF16PtrFromString(fsx.ExtendedPath(path))
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
	var id [fileIDInfoSize]byte
	if err := windows.GetFileInformationByHandleEx(h, windows.FileIdInfo, &id[0], fileIDInfoSize); err == nil {
		le := binary.LittleEndian
		return fileKey{le.Uint64(id[0:8]), le.Uint64(id[8:16]), le.Uint64(id[16:24])}, true
	}
	return fileKey{uint64(fi.VolumeSerialNumber), uint64(fi.FileIndexHigh)<<32 | uint64(fi.FileIndexLow), 0}, true
}
