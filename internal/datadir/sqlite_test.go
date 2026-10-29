package datadir

import "testing"

func TestReadOnlySQLiteURIKeepsTheDriveOutOfTheAuthority(t *testing.T) {
	if got := ReadOnlySQLiteURI("C:/Users/me/data.sqlite3"); got != "file:///C:/Users/me/data.sqlite3?mode=ro&immutable=1" {
		t.Fatalf("uri = %s", got)
	}
	if got := ReadOnlySQLiteURI("/home/me/data.sqlite3"); got != "file:///home/me/data.sqlite3?mode=ro&immutable=1" {
		t.Fatalf("uri = %s", got)
	}
}
