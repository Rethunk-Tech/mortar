package thunderstore

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestDepCompactorSharesIdenticalListsAndKeepsTheDependencies(t *testing.T) {
	t.Parallel()
	deps := []string{"BepInEx-BepInExPack-5.4.2100", "Owner-Lib-1.0.0"}
	names := depNames{index: map[string]uint32{}}
	mk := func(owner string) pkg {
		return pkg{Owner: owner, Versions: []version{{Number: "1.0.0", Deps: names.ids(deps)}, {Number: "1.0.1", Deps: names.ids(deps)}}}
	}
	c := newDepCompactor()
	a, b := mk("A"), mk("B")
	c.add(&a)
	c.add(&b)
	c.tab.names = names.names
	if len(names.names) != len(deps) {
		t.Errorf("names = %v, want each dependency once", names.names)
	}
	if &a.Versions[0].Deps[0] != &b.Versions[1].Deps[0] {
		t.Error("identical dependency lists must share one id slice")
	}
	if got := a.depsOf(a.Versions[1]); !slices.Equal(got, deps) || !c.known() {
		t.Errorf("deps = %v, want %v", got, deps)
	}
}

func TestAListingWhoseVersionNamesADependencyItDoesNotHoldIsRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "listing.json")
	body := `[{"owner":"A","name":"M","versions":[{"v":"1.0.0","d":[0,1]}]}]` + "\n" + `["Only-One-1.0.0"]`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadPackages("refused-listing", path); err == nil {
		t.Fatal("an index past the names must fail the load, not panic at the first lookup")
	}
}

func TestALoadedListingIsReleasedOnceIdleAndReadAgainOnTheNextUse(t *testing.T) {
	was := listingIdle
	listingIdle = 20 * time.Millisecond
	t.Cleanup(func() { listingIdle = was })
	const key = "idle-listing"
	path := filepath.Join(t.TempDir(), "listing.json")
	body := `[{"owner":"A","name":"M","versions":[{"v":"1.0.0","d":[0]}]}]` + "\n" + `["Only-One-1.0.0"]`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	held := func() bool {
		memoMu.Lock()
		defer memoMu.Unlock()
		_, ok := memo[key]
		return ok
	}
	if _, err := loadPackages(key, path); err != nil || !held() {
		t.Fatalf("load: %v, held = %v", err, held())
	}
	for deadline := time.Now().Add(5 * time.Second); held(); time.Sleep(5 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("the idle listing is still held")
		}
	}
	pk, err := loadPackages(key, path)
	if err != nil || len(pk) != 1 || !slices.Equal(pk[0].depsOf(pk[0].Versions[0]), []string{"Only-One-1.0.0"}) {
		t.Fatalf("reload = %+v, %v", pk, err)
	}
}
