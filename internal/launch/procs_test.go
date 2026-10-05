package launch

import "testing"

func TestRunsFrom(t *testing.T) {
	cases := []struct {
		name string
		p    Process
		want bool
	}{
		{"exe inside", Process{Exe: "/lib/common/G/G.bin"}, true},
		{"exe sibling", Process{Exe: "/lib/common/G2/G.bin"}, false},
		{"dotnet host, dll arg inside", Process{Exe: "/usr/bin/dotnet", Args: []string{"dotnet", "/lib/common/G/x.dll"}}, true},
		{"wine Z: arg", Process{Exe: "/proton/wine64", Args: []string{`Z:\lib\common\G\G.exe`}}, true},
		{"wine Z: other install", Process{Exe: "/proton/wine64", Args: []string{`Z:\other\G.exe`}}, false},
		{"nothing known", Process{}, false},
	}
	for _, c := range cases {
		if got := c.p.RunsFrom("/lib/common/G"); got != c.want {
			t.Errorf("%s = %v, want %v", c.name, got, c.want)
		}
	}
}
