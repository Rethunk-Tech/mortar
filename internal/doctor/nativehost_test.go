package doctor

import (
	"testing"

	"github.com/Rethunk-AI/mortar/internal/nxm"
)

func TestAppendNativeHostChecks(t *testing.T) {
	rows := []nxm.HostStatus{
		{Browser: "Google Chrome", State: nxm.HostOK},
		{Browser: "Firefox", State: nxm.HostMissing},
		{Browser: "Brave", State: nxm.HostStale},
	}
	checks := AppendNativeHostChecks(nil, rows)
	if len(checks) != 3 {
		t.Fatalf("len = %d", len(checks))
	}
	if checks[0].Status != Pass || checks[0].Fix != "" {
		t.Fatalf("ok: %+v", checks[0])
	}
	if checks[1].Status != Warn || checks[1].Fix != nativeHostFix {
		t.Fatalf("missing: %+v", checks[1])
	}
	if checks[2].Status != Warn || checks[2].Fix != nativeHostFix {
		t.Fatalf("stale: %+v", checks[2])
	}
	if checks[0].ID != "nativeHost:google-chrome" {
		t.Fatalf("id: %q", checks[0].ID)
	}
}

func TestAppendNativeHostChecksEmpty(t *testing.T) {
	if got := AppendNativeHostChecks([]Check{{ID: "x"}}, nil); len(got) != 1 {
		t.Fatalf("got %d", len(got))
	}
}

func TestAppendNativeHostChecksUnreadable(t *testing.T) {
	checks := AppendNativeHostChecks(nil, []nxm.HostStatus{{Browser: "Edge", State: nxm.HostUnreadable}})
	if len(checks) != 1 || checks[0].Fix != nativeHostFix {
		t.Fatalf("%+v", checks)
	}
}
