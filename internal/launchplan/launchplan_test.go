package launchplan

import (
	"errors"
	"testing"
)

func TestOneOverrideAndDedup(t *testing.T) {
	p := New(ModeProfile)
	if err := p.OverrideExe("skse", "skse64_loader.exe"); err != nil {
		t.Fatal(err)
	}
	if err := p.OverrideExe("other", "x.exe"); !errors.Is(err, ErrSecondOverride) {
		t.Fatalf("second override: %v", err)
	}
	if err := p.SetVersion("fabric", VersionManifest{ID: "f"}); !errors.Is(err, ErrSecondOverride) {
		t.Fatalf("version after exe: %v", err)
	}
	r := RuntimeReq{Kind: "dll-override", Key: "winhttp", Value: "native,builtin"}
	p.RequireRuntime(r)
	p.RequireRuntime(r)
	p.AddProcessName("game")
	p.AddProcessName("game")
	if len(p.RuntimeReqs) != 1 || len(p.ProcessNames) != 1 || p.Exe != "skse64_loader.exe" {
		t.Fatalf("plan %+v", p)
	}
}
