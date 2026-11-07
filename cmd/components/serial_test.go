package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestTheSerialIsTheSigningTimeSoTheCatalogCarriesItsOwnAge(t *testing.T) {
	t.Parallel()
	before := time.Now().Unix()
	serial, err := nextSerial(filepath.Join(t.TempDir(), "components.json"))
	if err != nil {
		t.Fatal(err)
	}
	if serial < uint64(before) || serial > uint64(time.Now().Unix()+1) { //nolint:gosec // G115: Unix times here are positive
		t.Fatalf("serial %d is not the signing time %d", serial, before)
	}
}
