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

func TestRestorePreviousPutsCommandAndDefaultIconBack(t *testing.T) {
	m := memReg{}
	m.register(`C:\Mortar\mortar.exe`)
	cmd := `"C:\Vortex\Vortex.exe" "%1"`
	icon := `C:\Vortex\Vortex.exe,0`
	m.restore(previousID(cmd, icon, "URL:Vortex"))
	if got := m[classKey][""]; got != "URL:Vortex" {
		t.Fatalf("class name = %q", got)
	}
	if got := m[commandKey][""]; got != cmd {
		t.Fatalf("command = %q", got)
	}
	if got := m[classKey+`\DefaultIcon`][""]; got != icon {
		t.Fatalf("DefaultIcon = %q", got)
	}
}

func TestRestorePreviousWithoutIconDropsMortarsDefaultIcon(t *testing.T) {
	m := memReg{}
	m.register(`C:\Mortar\mortar.exe`)
	m.restore(previousID(`"C:\Vortex\Vortex.exe" "%1"`, "", ""))
	if _, ok := m[classKey+`\DefaultIcon`]; ok {
		t.Fatalf("DefaultIcon still set: %v", m[classKey+`\DefaultIcon`])
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

func TestRestoreTwoLineValueKeepsTheName(t *testing.T) {
	m := memReg{}
	m.register(`C:\Mortar\mortar.exe`)
	m.restore("\"C:\\Vortex\\Vortex.exe\" \"%1\"\nC:\\Vortex\\Vortex.exe,0")
	if got := m[classKey+`\DefaultIcon`][""]; got != `C:\Vortex\Vortex.exe,0` {
		t.Fatalf("DefaultIcon = %q", got)
	}
	if got := m[classKey][""]; got != "URL:NXM Protocol" {
		t.Fatalf("class name = %q", got)
	}
}
