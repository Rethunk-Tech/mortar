package datadir

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestSharesAccountData(t *testing.T) {
	acct := t.TempDir()
	prev := accountHome
	accountHome = func() (string, error) { return acct, nil }
	t.Cleanup(func() { accountHome = prev })
	sandbox := t.TempDir()
	homeVar := "HOME"
	if runtime.GOOS == "windows" {
		homeVar = "USERPROFILE"
	}

	t.Setenv(homeVar, acct)
	if !SharesAccountData() {
		t.Fatal("the account's own HOME must count as its data")
	}

	t.Setenv(homeVar, sandbox)
	if runtime.GOOS == "windows" {
		t.Setenv("LOCALAPPDATA", filepath.Join(sandbox, "AppData", "Local"))
	} else {
		t.Setenv("XDG_DATA_HOME", "")
	}
	if SharesAccountData() {
		t.Fatal("a sandbox HOME must not count as the account's data")
	}

	if runtime.GOOS == "windows" {
		t.Setenv("LOCALAPPDATA", filepath.Join(acct, "AppData", "Local"))
	} else {
		t.Setenv("XDG_DATA_HOME", filepath.Join(acct, ".local", "share"))
	}
	if !SharesAccountData() {
		t.Fatal("a sandbox HOME whose data variable points at the account's data folder must count")
	}
}
