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

// amsiChunk is how much of a file one AmsiScanBuffer call sees.
const amsiChunk = 32 << 20

// amsi hands each file to the antivirus registered as the AMSI provider: Defender or a third party.
type amsi struct{}

func (amsi) Scan(ctx context.Context, dir string) (Detection, bool, error) {
	if err := amsiDLL.Load(); err != nil {
		return Detection{}, false, ErrNoScanner
	}
	app, err := windows.UTF16PtrFromString("Mortar")
	if err != nil {
		return Detection{}, false, err
	}
	var actx uintptr
	if hr, _, _ := amsiInitialize.Call(uintptr(unsafe.Pointer(app)), uintptr(unsafe.Pointer(&actx))); hr != 0 {
		return Detection{}, false, ErrNoScanner
	}
	defer func() { _, _, _ = amsiUninitialize.Call(actx) }()
	var session uintptr
	if hr, _, _ := amsiOpenSession.Call(actx, uintptr(unsafe.Pointer(&session))); hr != 0 {
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
			hr, _, _ := amsiScanBuffer.Call(actx, uintptr(unsafe.Pointer(&buf[0])), uintptr(n),
				uintptr(unsafe.Pointer(name)), session, uintptr(unsafe.Pointer(&result)))
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
