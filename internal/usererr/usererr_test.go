package usererr

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"syscall"
	"testing"
)

func TestKindTagged(t *testing.T) {
	err := Wrap(Busy, errors.New("stardew is running"))
	if KindOf(err) != Busy {
		t.Fatalf("Kind=%q", KindOf(err))
	}
	if !errors.Is(err, err) {
		t.Fatal("errors.Is self")
	}
	inner := errors.New("cause")
	wrapped := Wrap(NotFound, inner)
	if !errors.Is(wrapped, inner) {
		t.Fatal("errors.Is cause")
	}
	var ue *Error
	if !errors.As(wrapped, &ue) || ue.Kind() != NotFound {
		t.Fatal("errors.As")
	}
}

func TestKindStdlib(t *testing.T) {
	cases := []struct {
		err  error
		want Kind
	}{
		{fs.ErrNotExist, NotFound},
		{os.ErrNotExist, NotFound},
		{fs.ErrPermission, Permission},
		{os.ErrPermission, Permission},
		{context.DeadlineExceeded, Network},
		{syscall.ENOSPC, DiskFull},
		{&net.OpError{Op: "dial", Err: errors.New("refused")}, Network},
		{&net.DNSError{Name: "example", Err: "no such host"}, Network},
		{errors.New("plain"), Unknown},
	}
	for _, c := range cases {
		if got := KindOf(c.err); got != c.want {
			t.Errorf("%v: got %q want %q", c.err, got, c.want)
		}
	}
}

func TestErrorWireAndParse(t *testing.T) {
	err := Wrap(Damaged, errors.New("json: broken"))
	kind, raw := Parse(err.Error())
	if kind != Damaged || raw != "json: broken" {
		t.Fatalf("Parse(%q) = %q %q", err.Error(), kind, raw)
	}
	kind, raw = Parse("plain")
	if kind != Unknown || raw != "plain" {
		t.Fatalf("plain: %q %q", kind, raw)
	}
}

func TestWrapNil(t *testing.T) {
	if Wrap(Busy, nil) != nil {
		t.Fatal("nil")
	}
}

func TestMarshalCarriesTheKindOfAnUntaggedError(t *testing.T) {
	readOnly := &fs.PathError{Op: "open", Path: "profile.json", Err: syscall.EACCES}
	if got := string(Marshal(fmt.Errorf("save profile: %w", readOnly))); got != `{"kind":"permission"}` {
		t.Fatalf("read-only folder: %s", got)
	}
	// Nothing listens on the discard port, so the dial is refused the way an offline request is.
	var d net.Dialer
	_, derr := d.DialContext(t.Context(), "tcp", "127.0.0.1:9")
	refused := &url.Error{Op: "Get", URL: "https://api.nexusmods.com/v1/user/tracked_mods.json", Err: derr}
	if got := string(Marshal(fmt.Errorf("tracked mods: %w", refused))); got != `{"kind":"network"}` {
		t.Fatalf("refused connection: %s (%v)", got, derr)
	}
	if Marshal(errors.New("plain")) != nil {
		t.Fatal("an unknown error keeps the default cause")
	}
	if got := string(Marshal(Wrap(Busy, errors.New("running")))); got != `{"kind":"busy"}` {
		t.Fatalf("tagged: %s", got)
	}
}

type flagged struct {
	Key string `json:"key"`
}

func (flagged) Error() string { return "flagged" }
func (f flagged) Detail() any { return f }

func TestMarshalCarriesADetailFromTheChain(t *testing.T) {
	err := fmt.Errorf("add: %w", Wrap(Malware, flagged{Key: "k1"}))
	if got := string(Marshal(err)); got != `{"kind":"malware","detail":{"key":"k1"}}` {
		t.Fatalf("Marshal = %s", got)
	}
}
