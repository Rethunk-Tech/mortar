package lan

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-desktopenv/firewall"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestRulesBlock(t *testing.T) {
	t.Parallel()
	const exe = `C:\Users\Zoë\AppData\Local\Programs\Mortar\mortar.exe`
	rule := func(allow bool, profiles int32) firewall.Rule {
		return firewall.Rule{App: `c:\users\zoë\appdata\local\programs\mortar\MORTAR.exe`, Inbound: true, Allow: allow, Enabled: true, Profiles: profiles}
	}
	local, publicBlock := rule(true, firewall.ProfilesLocal), rule(false, firewall.ProfilePublic)
	for name, tc := range map[string]struct {
		rules   []firewall.Rule
		anyAddr bool
		want    bool
	}{
		"no rules":                     {nil, false, true},
		"allow only":                   {[]firewall.Rule{local}, false, true},
		"allow and public block":       {[]firewall.Rule{local, publicBlock}, false, false},
		"allow covers private only":    {[]firewall.Rule{rule(true, firewall.ProfilePrivate), publicBlock}, false, true},
		"windows public block counts":  {[]firewall.Rule{local, rule(false, firewall.ProfilePublic)}, false, false},
		"block on private":             {[]firewall.Rule{local, publicBlock, rule(false, firewall.ProfilePrivate)}, false, true},
		"block on all profiles":        {[]firewall.Rule{local, publicBlock, rule(false, firewall.ProfilesAll)}, false, true},
		"disabled block ignored":       {[]firewall.Rule{local, publicBlock, {App: exe, Inbound: true, Profiles: firewall.ProfilesAll}}, false, false},
		"other program ignored":        {[]firewall.Rule{{App: `C:\x.exe`, Inbound: true, Allow: true, Enabled: true, Profiles: firewall.ProfilesAll}}, false, true},
		"outbound allow ignored":       {[]firewall.Rule{{App: exe, Allow: true, Enabled: true, Profiles: firewall.ProfilesAll}}, false, true},
		"any: all-profile allow":       {[]firewall.Rule{rule(true, firewall.ProfilesAll)}, true, false},
		"any: local allow is short":    {[]firewall.Rule{local}, true, true},
		"any: public block disallowed": {[]firewall.Rule{rule(true, firewall.ProfilesAll), publicBlock}, true, true},
		"lan: all-profile allow only":  {[]firewall.Rule{rule(true, firewall.ProfilesAll)}, false, true},
	} {
		if got := firewall.Blocked(tc.rules, exe, tc.anyAddr); got != tc.want {
			t.Errorf("%s: Blocked = %v, want %v", name, got, tc.want)
		}
	}
}

func TestApplyDoesNotListenAtLaunchWithoutRules(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		blocked    bool
		wantListen bool
	}{"rules missing": {true, false}, "rules present": {false, true}} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			store, err := settings.OpenIn(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.Update(func(v *settings.Settings) { v.LanSharing, v.LanStarted = true, true }); err != nil {
				t.Fatal(err)
			}
			svc := NewService(Deps{Settings: store})
			svc.checkFirewall = func(context.Context, bool) (bool, error) { return tc.blocked, nil }
			svc.networkPublic = func() bool { return false }
			t.Cleanup(svc.Shutdown)
			if err := svc.Apply(); err != nil {
				if tc.wantListen && strings.Contains(err.Error(), "no multicast listeners") {
					t.Skip(err)
				}
				t.Fatal(err)
			}
			if got := svc.listening(); got != tc.wantListen {
				t.Errorf("listening = %v, want %v", got, tc.wantListen)
			}
			if got := store.Get().LanStarted; got != tc.wantListen {
				t.Errorf("lanStarted = %v, want %v", got, tc.wantListen)
			}
		})
	}
}

func TestOnlyANetworkSetToPublicAloneCountsAsPublic(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		current int32
		err     error
		want    bool
	}{
		"public only":        {firewall.ProfilePublic, nil, true},
		"private":            {firewall.ProfilePrivate, nil, false},
		"domain":             {firewall.ProfileDomain, nil, false},
		"public and private": {firewall.ProfilePublic | firewall.ProfilePrivate, nil, false},
		"public and domain":  {firewall.ProfilePublic | firewall.ProfileDomain, nil, false},
		"no network":         {0, nil, false},
		"unread public bits": {firewall.ProfilePublic, errors.New("read Windows Firewall profile"), false},
	} {
		if got := publicNetwork(tc.current, tc.err); got != tc.want {
			t.Errorf("%s: publicNetwork = %v, want %v", name, got, tc.want)
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
	if err := svc.ensureStarted(t.Context()); err == nil {
		t.Fatal("ensureStarted on a Public network = nil, want the Public-network error")
	}
	if _, err := svc.Peers(t.Context()); err == nil {
		t.Fatal("Peers on a Public network = nil error, want the Public-network error so `lan peers` can show it")
	}
	if svc.listening() {
		t.Fatal("listening on a Public network")
	}
	if !store.Get().LanStarted {
		t.Fatal("lanStarted was cleared; the rules still exist")
	}
}
