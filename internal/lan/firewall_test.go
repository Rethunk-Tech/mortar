package lan

import "testing"

func TestRulesBlock(t *testing.T) {
	t.Parallel()
	const exe = `C:\Users\Zoë\AppData\Local\Programs\Mortar\mortar.exe`
	allow := fwRule{App: exe, Inbound: true, Allow: true, Enabled: true, Profiles: profilePrivate}
	block := func(p int32) fwRule {
		return fwRule{App: `c:\users\zoë\appdata\local\programs\mortar\MORTAR.exe`, Inbound: true, Enabled: true, Profiles: p}
	}
	for name, tc := range map[string]struct {
		rules   []fwRule
		current int32
		want    bool
	}{
		"no rules":                     {nil, profilePrivate, true},
		"private allow":                {[]fwRule{allow}, profilePrivate, false},
		"allow on public only":         {[]fwRule{{App: exe, Inbound: true, Allow: true, Enabled: true, Profiles: 4}}, profilePrivate, true},
		"all-profile allow":            {[]fwRule{{App: exe, Inbound: true, Allow: true, Enabled: true, Profiles: profilesAll}}, profilePrivate, false},
		"block on current profile":     {[]fwRule{allow, block(profilePrivate)}, profilePrivate, true},
		"block on other profile":       {[]fwRule{allow, block(4)}, profilePrivate, false},
		"block on all profiles":        {[]fwRule{allow, block(profilesAll)}, profilePrivate, true},
		"disabled block ignored":       {[]fwRule{allow, {App: exe, Inbound: true, Profiles: profilesAll}}, profilePrivate, false},
		"other program's rule ignored": {[]fwRule{{App: `C:\x.exe`, Inbound: true, Allow: true, Enabled: true, Profiles: profilePrivate}}, profilePrivate, true},
		"outbound allow ignored":       {[]fwRule{{App: exe, Allow: true, Enabled: true, Profiles: profilePrivate}}, profilePrivate, true},
	} {
		if got := rulesBlock(tc.rules, exe, tc.current); got != tc.want {
			t.Errorf("%s: rulesBlock = %v, want %v", name, got, tc.want)
		}
	}
}
