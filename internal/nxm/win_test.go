package nxm

import "testing"

func TestPreviousRoundTrip(t *testing.T) {
	cmd, icon, name := `"C:\Vortex\Vortex.exe" "%1"`, `C:\Vortex\Vortex.exe,0`, "URL:Vortex"
	want := previous{cmd: cmd, icon: icon, name: name, hasIcon: true, hasName: true}
	if got := splitPrevious(previousID(cmd, icon, name)); got != want {
		t.Fatalf("splitPrevious = %+v, want %+v", got, want)
	}
	if got := splitPrevious(cmd + "\n" + icon); got != (previous{cmd: cmd, icon: icon, hasIcon: true}) {
		t.Fatalf("two-line value = %+v", got)
	}
	if previousID("", icon, name) != "" || splitPrevious("") != (previous{}) {
		t.Fatal("no command means nothing saved")
	}
}
