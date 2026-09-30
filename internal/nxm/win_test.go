package nxm

import "testing"

func TestRegisterWritesDefaultIconAtExe(t *testing.T) {
	exe := `C:\Program Files\Mortar\mortar.exe`
	m := memReg{}
	m.register(exe)
	if got := m[classKey+`\DefaultIcon`][""]; got != exe+",0" {
		t.Fatalf("DefaultIcon = %q", got)
	}
	if got := m[commandKey][""]; got != `"`+exe+`" "%1"` {
		t.Fatalf("command = %q", got)
	}
}

func TestRestorePreviousKeepsTheHandlerCommand(t *testing.T) {
	m := memReg{}
	m.register(`C:\Mortar\mortar.exe`)
	prev := `"C:\Vortex\Vortex.exe" "%1"`
	m.restore(prev)
	if got := m[commandKey][""]; got != prev {
		t.Fatalf("command = %q", got)
	}
}

func TestRestoreWithoutPreviousRemovesTheClass(t *testing.T) {
	m := memReg{}
	m.register(`C:\Mortar\mortar.exe`)
	m.restore("")
	if len(m) != 0 {
		t.Fatalf("keys left: %v", m)
	}
}
