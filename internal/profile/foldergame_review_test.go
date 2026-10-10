package profile

import (
	"slices"
	"testing"
)

func TestFolderGameBackupCarriesTheStoreItemOfAPerFileEntry(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, _ := e.Create(folderGame, "S")
	zip := zipOf(t, "a.zip", map[string]string{"one.package": "1", "two.package": "2"})
	res, err := e.InstallSource(t.Context(), folderGame, p.ID, zip, cfSource(10))
	if err != nil || len(res.Profile.Entries) != 2 {
		t.Fatalf("%v %v", keysOf(res.Profile), err)
	}
	item := res.Profile.Entries[0].Item
	dirs, err := e.BackupDirs(folderGame, res.Profile, func(Entry) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := dirs["store/"+item+"/"]; !ok {
		t.Fatalf("backup omits the archive %q the entries are cut from: %v", item, dirs)
	}
}

func TestFolderGamePackagesListPerFileEntries(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, _ := e.Create(folderGame, "S")
	zip := zipOf(t, "a.zip", map[string]string{"one.package": "1", "two.package": "2"})
	if _, err := e.InstallSource(t.Context(), folderGame, p.ID, zip, cfSource(10)); err != nil {
		t.Fatal(err)
	}
	refs, err := e.Packages(folderGame, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(refs, func(r PackageRef) bool { return r.Name != "" }) || len(refs) != 2 {
		t.Fatalf("packages = %+v", refs)
	}
}
