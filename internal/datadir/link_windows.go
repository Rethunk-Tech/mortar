package datadir

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"unicode/utf16"

	"golang.org/x/sys/windows"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// mountPointTag is IO_REPARSE_TAG_MOUNT_POINT, the reparse tag of a directory junction.
const mountPointTag = 0xA0000003

// maxJunctionPath keeps both names and their header inside the 16-bit lengths a mount point stores.
const maxJunctionPath = 8000

var errPathTooLong = errors.New("the folder path is too long for a junction")

const (
	maxU16 = 0xFFFF
	maxU32 = 0xFFFFFFFF
)

// LinkDir makes link point at the folder target: a symbolic link when the account may create one, else a directory
// junction, which needs no privilege.
func LinkDir(link, target string) error {
	if os.Symlink(target, link) == nil {
		return nil
	}
	return makeJunction(link, target)
}

// makeJunction creates the empty folder link and turns it into a junction to target by setting its reparse point.
func makeJunction(link, target string) error {
	abs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if err := os.Mkdir(link, 0o700); err != nil {
		return err
	}
	if err := setMountPoint(link, abs); err != nil {
		_ = fsx.Remove(link)
		return err
	}
	return nil
}

func setMountPoint(link, target string) error {
	path, err := windows.UTF16PtrFromString(link)
	if err != nil {
		return err
	}
	h, err := windows.CreateFile(path, windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(h) }()

	sub := utf16.Encode([]rune(`\??\` + target))
	shown := utf16.Encode([]rune(target))
	if len(target) > maxJunctionPath {
		return errPathTooLong
	}
	names := make([]byte, 0, (len(sub)+len(shown)+2)*2)
	for _, u := range append(append(sub, 0), append(shown, 0)...) {
		names = binary.LittleEndian.AppendUint16(names, u)
	}
	// REPARSE_DATA_BUFFER for a mount point: tag, data length, reserved, then four name offsets and lengths, then the
	// substitute name and the print name, each ending in a NUL.
	buf := binary.LittleEndian.AppendUint32(nil, mountPointTag)
	buf = binary.LittleEndian.AppendUint16(buf, uint16((8+len(names))&maxU16))
	buf = binary.LittleEndian.AppendUint16(buf, 0)
	buf = binary.LittleEndian.AppendUint16(buf, 0)
	buf = binary.LittleEndian.AppendUint16(buf, uint16((len(sub)*2)&maxU16))
	buf = binary.LittleEndian.AppendUint16(buf, uint16((len(sub)*2+2)&maxU16))
	buf = binary.LittleEndian.AppendUint16(buf, uint16((len(shown)*2)&maxU16))
	buf = append(buf, names...)
	var returned uint32
	return windows.DeviceIoControl(h, windows.FSCTL_SET_REPARSE_POINT, &buf[0], uint32(len(buf)&maxU32), nil, 0, &returned, nil)
}
