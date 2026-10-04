package appversion

import (
	"os"
	"testing"
)

func TestFromConfigReadsInfoVersion(t *testing.T) {
	got, err := FromConfig([]byte("version: '3'\n\ninfo:\n  productName: \"Mortar\"\n  version: \"1.2.3\" # comment\n"))
	if err != nil || got != "1.2.3" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := FromConfig([]byte("version: '3'\n")); err == nil {
		t.Fatal("no info.version should be an error")
	}
}

func TestRepoConfigHasAVersion(t *testing.T) {
	b, err := os.ReadFile("../../build/config.yml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := FromConfig(b); err != nil {
		t.Fatal(err)
	}
}
