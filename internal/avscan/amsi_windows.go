//go:build windows

package avscan

import (
	"context"
	"errors"
	"fmt"
	"io"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

var (
	amsiDLL          = windows.NewLazySystemDLL("amsi.dll")
	amsiInitialize   = amsiDLL.NewProc("AmsiInitialize")
	amsiUninitialize = amsiDLL.NewProc("AmsiUninitialize")
	amsiOpenSession  = amsiDLL.NewProc("AmsiOpenSession")
	amsiCloseSession = amsiDLL.NewProc("AmsiCloseSession")
	amsiScanBuffer   = amsiDLL.NewProc("AmsiScanBuffer")
)

// amsiDetected is AMSI_RESULT_DETECTED: results at or above it are malware (AmsiResultIsMalware).
const amsiDetected = 32768

// amsiNotReady is HRESULT_FROM_WIN32(ERROR_NOT_READY), which AmsiScanBuffer returns when no antivirus is registered
// as the AMSI provider.
const amsiNotReady = 0x80070015

// amsiChunk is how much of a file one AmsiScanBuffer call sees.
const amsiChunk = 32 << 20

func amsiPtr[T any](v *T) uintptr {
	return uintptr(unsafe.Pointer(v)) //nolint:gosec // G103: AMSI takes pointers to Go values
}

// amsi hands each file to the antivirus registered as the AMSI provider: Defender or a third party.
type amsi struct{}

func (amsi) Scan(ctx context.Context, dir string) (Detection, bool, error) {
	if err := amsiDLL.Load(); err != nil {
		return Detection{}, false, fmt.Errorf("%w: amsi.dll: %w", ErrNoScanner, err)
	}
	app, err := windows.UTF16PtrFromString("Mortar")
	if err != nil {
		return Detection{}, false, err
	}
	var actx uintptr
	if hr, _, _ := amsiInitialize.Call(amsiPtr(app), amsiPtr(&actx)); hr != 0 {
		return Detection{}, false, fmt.Errorf("%w: AmsiInitialize HRESULT 0x%x", ErrNoScanner, hr)
	}
	defer func() { _, _, _ = amsiUninitialize.Call(actx) }()
	var session uintptr
	if hr, _, _ := amsiOpenSession.Call(actx, amsiPtr(&session)); hr != 0 {
		return Detection{}, false, fmt.Errorf("AmsiOpenSession: HRESULT 0x%x", hr)
	}
	defer func() { _, _, _ = amsiCloseSession.Call(actx, session) }()
	var found Detection
	var hit bool
	errStop := errors.New("stop")
	err = files(ctx, dir, func(abs, rel string) error {
		flagged, err := scanFileAMSI(ctx, actx, session, abs, rel)
		if err != nil {
			return err
		}
		if flagged {
			found, hit = Detection{Name: "malware reported by the Windows antivirus", File: rel, Scanner: "AMSI"}, true
			return errStop
		}
		return nil
	})
	if err != nil && !errors.Is(err, errStop) {
		return Detection{}, false, err
	}
	return found, hit, nil
}

func scanFileAMSI(ctx context.Context, actx, session uintptr, path, rel string) (bool, error) {
	in, err := fsx.Open(path)
	if errors.Is(err, windows.ERROR_VIRUS_INFECTED) || errors.Is(err, windows.ERROR_VIRUS_DELETED) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	defer func() { _ = in.Close() }()
	name, err := windows.UTF16PtrFromString(rel)
	if err != nil {
		return false, err
	}
	buf := make([]byte, amsiChunk)
	for {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		n, readErr := io.ReadFull(in, buf)
		if n > 0 {
			var result uint32
			hr, _, _ := amsiScanBuffer.Call(actx, amsiPtr(&buf[0]), uintptr(n),
				amsiPtr(name), session, amsiPtr(&result))
			if hr&0xFFFFFFFF == amsiNotReady {
				return false, fmt.Errorf("%w: AmsiScanBuffer: no provider answered", ErrNoScanner)
			}
			if hr != 0 {
				return false, fmt.Errorf("AmsiScanBuffer: HRESULT 0x%x", hr)
			}
			if result >= amsiDetected {
				return true, nil
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) || errors.Is(readErr, io.ErrUnexpectedEOF) {
				return false, nil
			}
			return false, readErr
		}
	}
}
