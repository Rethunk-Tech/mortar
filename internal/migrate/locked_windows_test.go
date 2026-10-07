package migrate

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// Vortex opens its write-ahead log without read sharing; reading the database meanwhile must say so, and the other
// managers' profiles must still be listed.
func TestRunningVortexIsReportedNotHidden(t *testing.T) {
	home, state, _ := newVortexHome(t, "")
	wal := filepath.Join(state, "000099.log")
	if err := os.WriteFile(wal, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	name, err := syscall.UTF16PtrFromString(wal)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(name, syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = syscall.CloseHandle(h) }()
	if _, err := VortexContents(home, ""); !errors.Is(err, ErrVortexRunning) {
		t.Fatalf("VortexContents error = %v", err)
	}
	sources, err := Detect(home, t.TempDir(), "stardew", "")
	if err != nil || len(sources) != 1 || sources[0].Kind != KindVortex || sources[0].Error != ErrVortexRunning.Error() {
		t.Fatalf("sources = %#v, %v", sources, err)
	}
}
