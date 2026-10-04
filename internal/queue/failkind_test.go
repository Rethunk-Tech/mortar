package queue

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/nexus"
	"github.com/Rethunk-AI/mortar/internal/store"
)

func TestFailureKind(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{fmt.Errorf("x: %w", nexus.ErrUnauthorized), FailAuth},
		{nexus.ErrQuarantined, FailBlocked},
		{&store.DiskFullError{NeedMB: 5}, FailDisk},
		{&url.Error{Op: "Get", URL: "https://x", Err: &net.OpError{Op: "dial", Err: errors.New("refused")}}, FailNetwork},
		{fmt.Errorf("read: %w", io.ErrUnexpectedEOF), FailNetwork},
		{errors.New("hash mismatch"), FailOther},
	}
	for _, c := range cases {
		if got := failureKind(c.err); got != c.want {
			t.Errorf("%v: got %q want %q", c.err, got, c.want)
		}
	}
}
