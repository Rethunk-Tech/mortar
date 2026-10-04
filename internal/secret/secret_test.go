package secret

import (
	"errors"
	"strings"
	"testing"
)

func TestExplainMapsMissingProvider(t *testing.T) {
	raw := errors.New(`The name org.freedesktop.secrets was not provided by any .service files: org.freedesktop.DBus.Error.ServiceUnknown`)
	got := explain(raw).Error()
	if !strings.Contains(got, "No keyring service is running") || strings.Contains(got, "DBus") {
		t.Fatalf("got %q", got)
	}
	other := errors.New("boom")
	if !errors.Is(explain(other), other) || explain(nil) != nil {
		t.Fatal("unrelated errors must pass through")
	}
}
