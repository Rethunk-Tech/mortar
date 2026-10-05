package queue

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"slices"
	"testing"
)

func TestVerifyDigestSHA512(t *testing.T) {
	sum := sha512.Sum512([]byte("abc"))
	p := writeFile(t, "abc")
	if err := verifyDigest(p, "SHA512:"+hex.EncodeToString(sum[:])); err != nil {
		t.Errorf("matching digest: %v", err)
	}
	if err := verifyDigest(p, "sha512:00"); err == nil {
		t.Error("a wrong digest passed")
	}
}

func TestExpandDirectQueuesDependenciesFirstOnce(t *testing.T) {
	files := map[string]DirectFile{
		"root": {Version: "2", URL: "u/root", Digest: "sha512:aa", Dependencies: []DirectRef{{ID: "a", Version: "1"}, {ID: "b"}}},
		"a":    {Version: "1", URL: "u/a", Dependencies: []DirectRef{{ID: "b"}}},
		"b":    {Version: "5", URL: "u/b"},
	}
	s := &Service{d: Deps{Direct: func(_ context.Context, src, id, _ string, _ []string) (DirectFile, error) {
		if src != "modrinth" {
			t.Errorf("source %q", src)
		}
		return files[id], nil
	}}}
	got, err := s.expandPackages(context.Background(), []Request{{Kind: KindInstall, Game: "g", Profile: "p", Source: "modrinth", Package: "root", Name: "Root"}})
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, r := range got {
		order = append(order, r.Package)
	}
	if !slices.Equal(order, []string{"b", "a", "root"}) {
		t.Fatalf("order %v", order)
	}
	if got[0].Kind != KindDependency || got[2].Kind != KindInstall || got[2].Name != "Root" || got[2].digest != "sha512:aa" || got[2].url != "u/root" {
		t.Fatalf("%+v", got)
	}
}
