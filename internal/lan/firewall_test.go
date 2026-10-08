package lan

import "testing"

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
