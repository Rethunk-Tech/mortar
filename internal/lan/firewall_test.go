package lan

import (
	"context"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestRulesBlock(t *testing.T) {
	t.Parallel()
	const exe = `C:\Users\Zoë\AppData\Local\Programs\Mortar\mortar.exe`
	rule := func(allow bool, profiles int32) fwRule {
		return fwRule{App: `c:\users\zoë\appdata\local\programs\mortar\MORTAR.exe`, Inbound: true, Allow: allow, Enabled: true, Profiles: profiles}
	}
	local, publicBlock := rule(true, profilesLocal), rule(false, profilePublic)
	for name, tc := range map[string]struct {
		rules   []fwRule
		anyAddr bool
		want    bool
	}{
		"no rules":                     {nil, false, true},
		"allow only":                   {[]fwRule{local}, false, true},
		"allow and public block":       {[]fwRule{local, publicBlock}, false, false},
		"allow covers private only":    {[]fwRule{rule(true, profilePrivate), publicBlock}, false, true},
		"windows public block counts":  {[]fwRule{local, rule(false, profilePublic)}, false, false},
		"block on private":             {[]fwRule{local, publicBlock, rule(false, profilePrivate)}, false, true},
		"block on all profiles":        {[]fwRule{local, publicBlock, rule(false, profilesAll)}, false, true},
		"disabled block ignored":       {[]fwRule{local, publicBlock, {App: exe, Inbound: true, Profiles: profilesAll}}, false, false},
		"other program ignored":        {[]fwRule{{App: `C:\x.exe`, Inbound: true, Allow: true, Enabled: true, Profiles: profilesAll}}, false, true},
		"outbound allow ignored":       {[]fwRule{{App: exe, Allow: true, Enabled: true, Profiles: profilesAll}}, false, true},
		"any: all-profile allow":       {[]fwRule{rule(true, profilesAll)}, true, false},
		"any: local allow is short":    {[]fwRule{local}, true, true},
		"any: public block disallowed": {[]fwRule{rule(true, profilesAll), publicBlock}, true, true},
		"lan: all-profile allow only":  {[]fwRule{rule(true, profilesAll)}, false, true},
	} {
		if got := rulesBlock(tc.rules, exe, tc.anyAddr); got != tc.want {
			t.Errorf("%s: rulesBlock = %v, want %v", name, got, tc.want)
		}
	}
}

func TestApplyDoesNotListenAtLaunchWithoutRules(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		blocked    bool
		wantListen bool
	}{"rules missing": {true, false}, "rules present": {false, true}} {
		store, err := settings.OpenIn(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.Update(func(v *settings.Settings) { v.LanSharing, v.LanStarted = true, true }); err != nil {
			t.Fatal(err)
		}
		svc := NewService(Deps{Settings: store})
		svc.checkFirewall = func(context.Context, bool) (bool, error) { return tc.blocked, nil }
		t.Cleanup(svc.Shutdown)
		if err := svc.Apply(); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := svc.listening(); got != tc.wantListen {
			t.Errorf("%s: listening = %v, want %v", name, got, tc.wantListen)
		}
		if got := store.Get().LanStarted; got != tc.wantListen {
			t.Errorf("%s: lanStarted = %v, want %v", name, got, tc.wantListen)
		}
	}
}

func TestListenBlockedByPublic(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		current int32
		anyAddr bool
		want    bool
	}{
		"public only":        {profilePublic, false, true},
		"public only, any":   {profilePublic, true, false},
		"private":            {profilePrivate, false, false},
		"domain":             {profileDomain, false, false},
		"public and private": {profilePublic | profilePrivate, false, false},
		"public and domain":  {profilePublic | profileDomain, false, false},
		"no network":         {0, false, false},
	} {
		if got := listenBlockedByPublic(tc.current, tc.anyAddr); got != tc.want {
			t.Errorf("%s: listenBlockedByPublic = %v, want %v", name, got, tc.want)
		}
	}
}

func TestPublicNetworkWithholdsListening(t *testing.T) {
	t.Parallel()
	store, err := settings.OpenIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(func(v *settings.Settings) { v.LanSharing, v.LanStarted = true, true }); err != nil {
		t.Fatal(err)
	}
	svc := NewService(Deps{Settings: store})
	svc.checkFirewall = func(context.Context, bool) (bool, error) { return false, nil }
	svc.networkPublic = func() bool { return true }
	t.Cleanup(svc.Shutdown)
	if err := svc.Apply(); err == nil {
		t.Fatal("Apply on a Public network = nil, want the Public-network error")
	}
	if err := svc.ensureStarted(context.Background()); err == nil {
		t.Fatal("ensureStarted on a Public network = nil, want the Public-network error")
	}
	if svc.listening() {
		t.Fatal("listening on a Public network")
	}
	if !store.Get().LanStarted {
		t.Fatal("lanStarted was cleared; the rules still exist")
	}
}
