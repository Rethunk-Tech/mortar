package controlwire

import (
	"errors"
	"testing"
	"time"
)

func TestCallDirNotRunning(t *testing.T) {
	if err := CallDir(t.TempDir(), "hello", nil, nil, time.Second); !errors.Is(err, ErrNotRunning) {
		t.Errorf("empty dir: %v, want ErrNotRunning", err)
	}
	if err := CallDir(t.TempDir()+"/gone", "hello", nil, nil, time.Second); !errors.Is(err, ErrNotRunning) {
		t.Errorf("missing dir: %v, want ErrNotRunning", err)
	}
}

func TestCallDirDecodesResultAndChecksProtocol(t *testing.T) {
	dir, _ := liveServer(t, "tok")
	var h Hello
	if err := CallDir(dir, "hello", nil, &h, 2*time.Second); err != nil || h.Protocol != Protocol {
		t.Fatalf("CallDir = %+v, %v", h, err)
	}
	if err := CheckProtocol(dir, 2*time.Second); err != nil {
		t.Errorf("CheckProtocol: %v", err)
	}
	// liveServer drops a connection that asks anything but hello.
	if err := CallDir(dir, "other", nil, nil, 2*time.Second); err == nil {
		t.Error("an unanswered method succeeded")
	}
}

func TestUnansweringFalseWhenAppAnswers(t *testing.T) {
	dir, _ := liveServer(t, "tok")
	if pid, ok := Unanswering(dir, 0); ok || pid != 0 {
		t.Errorf("Unanswering = %d, %v for a live app", pid, ok)
	}
}

func TestRemoteErrorText(t *testing.T) {
	var err error = &RemoteError{Msg: "boom", Kind: "k"}
	if re := new(RemoteError); !errors.As(err, &re) || re.Error() != "boom" {
		t.Errorf("RemoteError text = %q", err)
	}
}
