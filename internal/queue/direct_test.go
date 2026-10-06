package queue

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source/curseforge"
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

func TestExpandDirectOpensThePageOfAForbiddenFile(t *testing.T) {
	var opened string
	s := &Service{d: Deps{
		OpenURL: func(u string) error { opened = u; return nil },
		Direct: func(context.Context, string, string, string, []string) (DirectFile, error) {
			return DirectFile{}, &curseforge.NotDistributableError{Mod: "Shut", PageURL: "https://www.curseforge.com/projects/20"}
		},
	}}
	_, err := s.expandPackages(context.Background(), []Request{{Kind: KindInstall, Game: "g", Profile: "p", Source: "curseforge", Package: "20"}})
	if err == nil || opened != "https://www.curseforge.com/projects/20" || !strings.Contains(err.Error(), "Shut") {
		t.Fatalf("opened %q, err %v", opened, err)
	}
}

func TestExpandDirectPinsCurseForgeFileID(t *testing.T) {
	var asked string
	s := &Service{d: Deps{Direct: func(_ context.Context, _, _, version string, _ []string) (DirectFile, error) {
		asked = version
		return DirectFile{Version: "Content Patcher 2.9.1", FileID: 777, URL: "u"}, nil
	}}}
	got, err := s.expandDirect(context.Background(), Request{Kind: KindInstall, Source: "curseforge", Package: "309243", Version: "Content Patcher 2.9.1", PackageFile: 555})
	if err != nil || len(got) != 1 {
		t.Fatalf("got %v, err %v", got, err)
	}
	if asked != "555" {
		t.Errorf("resolved %q, want the exact file id 555", asked)
	}
	if got[0].PackageFile != 777 || got[0].Version != "Content Patcher 2.9.1" {
		t.Errorf("request carries file %d version %q", got[0].PackageFile, got[0].Version)
	}
	if src := packageSource(Item{Source: "curseforge", Package: "309243", Version: got[0].Version, PackageFile: got[0].PackageFile}); src.FileID != 777 {
		t.Errorf("source file id %d", src.FileID)
	}
}
