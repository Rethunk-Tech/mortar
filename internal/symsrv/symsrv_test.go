package symsrv

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// testdata/UnityPlayer.dll and its PDB are built by testdata/src/build.sh: DrawBuffers at RVA 0x1000, UnityMain at
// 0x1030. llvm-pdbutil prints the PDB's GUID as {4FABE818-577A-B230-4C4C-44205044422E}, age 1, which is the
// server key once its first three fields are read little-endian.
const fixturePDB = "UnityPlayer_Win64_player_mono_x64.pdb"

func fixture(t *testing.T) CodeView {
	t.Helper()
	cv, ok := PECodeView(filepath.Join("testdata", "UnityPlayer.dll"))
	if !ok || cv.PDB != fixturePDB || cv.Key() != "4FABE818577AB2304C4C44205044422E1" {
		t.Fatalf("CodeView = %+v (key %s)", cv, cv.Key())
	}
	return cv
}

func TestLookupNamesTheNearestPublicFunction(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", fixturePDB))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	info, _ := f.Stat()
	for _, c := range []struct {
		rva    uint64
		name   string
		offset uint64
	}{{0x101c, "GfxDeviceD3D11::DrawBuffers", 0x1c}, {0x1050, "UnityMain", 0x20}, {0x10, "", 0}} {
		name, offset, err := Lookup(f, info.Size(), c.rva)
		if name != c.name || offset != c.offset || (err == nil) != (c.name != "") {
			t.Errorf("%#x: %q +%#x, %v", c.rva, name, offset, err)
		}
	}
	for in, want := range map[string]string{"??0Gfx@Unity@@QEAA@XZ": "Unity::Gfx::Gfx", "??1Gfx@@QEAA@XZ": "Gfx::~Gfx", "??_GFoo@@UEAAPEAXI@Z": "??_GFoo@@UEAAPEAXI@Z"} {
		if got := undecorate(in); got != want {
			t.Errorf("undecorate(%s) = %s", in, got)
		}
	}
}

func TestURLsRouteByPDBName(t *testing.T) {
	cv := fixture(t)
	unity := URLs(cv)
	if len(unity) != 2 || unity[0] != "https://symbolserver.unity3d.com/"+fixturePDB+"/4FABE818577AB2304C4C44205044422E1/"+fixturePDB || !strings.HasPrefix(unity[1], "http://") {
		t.Errorf("Unity's PDB: %v", unity)
	}
	for name, server := range map[string]string{"NTDLL.pdb": "https://msdl.microsoft.com/", "vcruntime140_1.amd64.pdb": "https://msdl.microsoft.com/", "Cool.pdb": "", "game.pdb": "", "../ntdll.pdb": ""} {
		got := URLs(CodeView{PDB: name, GUID: cv.GUID, Age: cv.Age})
		if server == "" && len(got) != 0 || server != "" && (len(got) != 1 || !strings.HasPrefix(got[0], server)) {
			t.Errorf("%s: %v", name, got)
		}
	}
}

// A PDB is fetched once and then read from the cache, a 404 is remembered, and nothing is fetched while offline.
func TestResolverFetchesOnceAndCaches(t *testing.T) {
	cv := fixture(t)
	pdb, err := os.ReadFile(filepath.Join("testdata", fixturePDB))
	if err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/"+fixturePDB+"/"+cv.Key()+"/"+fixturePDB {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(pdb)
	}))
	defer srv.Close()
	saved := unityServers
	unityServers = []string{srv.URL + "/"}
	t.Cleanup(func() { unityServers = saved })
	online := true
	r := &Resolver{HTTP: srv.Client(), Dir: t.TempDir(), Online: func() bool { return online }, Wait: 10 * time.Second}
	dir := filepath.Join(r.Dir, cv.PDB, cv.Key())
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, cv.PDB+".part"), []byte("left by a crash"), 0o600); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if name, off, ok := r.Name(context.Background(), cv, 0x101c); !ok || name != "GfxDeviceD3D11::DrawBuffers" || off != 0x1c {
			t.Fatalf("%q +%#x %v", name, off, ok)
		}
	}
	other := cv
	other.Age = 2
	for range 2 {
		if _, _, ok := r.Name(context.Background(), other, 0x101c); ok {
			t.Fatal("a PDB the server lacks named a symbol")
		}
	}
	online = false
	other.Age = 3
	if _, _, ok := r.Name(context.Background(), other, 0x101c); ok || requests.Load() != 2 {
		t.Fatalf("requests = %d, want one for the PDB and one for the miss", requests.Load())
	}
}

// A miss marker is retried after missMaxAge, and prune clears expired markers and the folders they leave empty.
func TestMissExpires(t *testing.T) {
	r := &Resolver{Dir: t.TempDir()}
	old := time.Now().Add(-missMaxAge - time.Hour)
	stale := filepath.Join(r.Dir, "a.pdb", "KEY1")
	fresh := filepath.Join(r.Dir, "b.pdb", "KEY2")
	for _, d := range []string{stale, fresh} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, missMarker), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chtimes(filepath.Join(stale, missMarker), old, old); err != nil {
		t.Fatal(err)
	}
	if freshMiss(stale) || !freshMiss(fresh) {
		t.Fatalf("freshMiss: stale %v, fresh %v", freshMiss(stale), freshMiss(fresh))
	}
	r.prune()
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	r.prune()
	if _, err := os.Stat(filepath.Join(r.Dir, "a.pdb")); !os.IsNotExist(err) {
		t.Fatalf("stale miss folders remain: %v", err)
	}
	if !freshMiss(fresh) {
		t.Fatal("a fresh miss was pruned")
	}
}
