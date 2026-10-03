package usererr

import (
	"context"
	"errors"
	"io/fs"
	"net"
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
