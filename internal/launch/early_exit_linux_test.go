//go:build linux

package launch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// runFake runs script as the launched process, a stand-in for a Proton or Wine runtime.
func runFake(t *testing.T, script string) (time.Duration, error) {
	t.Helper()
	began := time.Now()
	err := Run(context.Background(), Start, Command{Name: "sh", Args: []string{"-c", script}}, Timing{Timeout: 10 * time.Second, Poll: 10 * time.Millisecond}, nil)
	return time.Since(began), err
}

func TestAnEarlyExitSaysWhyAtOnce(t *testing.T) {
	for _, tc := range []struct {
		name, script string
		hint         Hint
		line         string
	}{
		{
			"steamclient does not load", `echo '0024:err:steamclient:steamclient_init unable to load native steamclient library' >&2; exit 1`,
			HintSteamClient, "unable to load native steamclient library",
		},
		{
			"wine assertion", `echo '0024:err:msvcrt:_wassert (L"!status",L"steamclient_main.c",375)' >&2; exit 3`,
			HintWine, "_wassert",
		},
		{
			"executable not found", `echo '/games/Lethal Company.exe: No such file or directory' >&2; exit 127`,
			HintMissingExe, "No such file or directory",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			took, err := runFake(t, tc.script)
			var f *Failure
			if !errors.As(err, &f) || f.Hint != tc.hint || !strings.Contains(f.Error(), tc.line) {
				t.Fatalf("err = %v (%T), want a %q failure naming %q", err, err, tc.hint, tc.line)
			}
			if took > 5*time.Second {
				t.Fatalf("took %v, want the failure at once", took)
			}
		})
	}
}

func TestAnExitWithNoKnownCauseCarriesCodeAndLastLines(t *testing.T) {
	_, err := runFake(t, `echo one >&2; echo two >&2; exit 4`)
	var e *ExitError
	if !errors.As(err, &e) || e.Code != 4 || strings.Join(e.Output, "|") != "one|two" {
		t.Fatalf("err = %#v", err)
	}
	if !strings.Contains(e.Error(), "code 4") || !strings.Contains(e.Error(), "two") {
		t.Fatalf("message = %q", e.Error())
	}
}

func TestAStalledLaunchIsEndedWithItsChildren(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	// The assertion dialog case: the runtime reports the error and then waits forever on a child it started.
	script := `sleep 60 & echo $! >` + pidFile + `; echo '0024:err:msvcrt:_wassert (L"!status")' >&2; wait`
	took, err := runFake(t, script)
	var f *Failure
	if !errors.As(err, &f) || f.Hint != HintWine {
		t.Fatalf("err = %v", err)
	}
	if took > 5*time.Second {
		t.Fatalf("took %v, want the failure before the timeout", took)
	}
	raw, readErr := fsx.ReadFile(pidFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		// A killed child is reaped by init; until then signal 0 still finds a zombie, so read its state instead.
		stat, statErr := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
		if statErr != nil || strings.Contains(string(stat), ") Z") {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
	t.Fatalf("child %d outlived the failed launch", pid)
}
