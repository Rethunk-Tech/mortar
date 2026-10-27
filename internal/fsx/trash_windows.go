package fsx

import (
	"fmt"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

var shFileOperation = windows.NewLazySystemDLL("shell32.dll").NewProc("SHFileOperationW")

// shFileOpStruct is SHFILEOPSTRUCTW.
type shFileOpStruct struct {
	hwnd          uintptr
	wFunc         uint32
	from          *uint16
	to            *uint16
	flags         uint16
	aborted       int32
	nameMappings  uintptr
	progressTitle *uint16
}

const (
	foDelete          = 3
	fofSilent         = 0x4
	fofNoConfirmation = 0x10
	fofAllowUndo      = 0x40
	fofNoErrorUI      = 0x400
)

// Trash moves path to the Recycle Bin, where Explorer can restore it.
func Trash(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	from, err := windows.UTF16FromString(abs)
	if err != nil {
		return err
	}
	// The list of paths ends with a second NUL.
	from = append(from, 0)
	op := shFileOpStruct{wFunc: foDelete, from: &from[0], flags: fofSilent | fofNoConfirmation | fofAllowUndo | fofNoErrorUI}
	r, _, _ := shFileOperation.Call(uintptr(unsafe.Pointer(&op))) //nolint:gosec // G103: SHFileOperation takes a pointer to the op struct
	if r != 0 || op.aborted != 0 {
		return fmt.Errorf("moving %s to the Recycle Bin failed (code %d)", abs, r)
	}
	return nil
}
