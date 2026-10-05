package control

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/pack"
	"github.com/Rethunk-Tech/mortar/internal/packsvc"
	"github.com/Rethunk-Tech/mortar/internal/queue"
)

type recordQueue struct{ got []queue.Request }

func (q *recordQueue) Add(_ context.Context, reqs []queue.Request) ([]queue.Item, error) {
	q.got = append(q.got, reqs...)
	return make([]queue.Item, len(reqs)), nil
}

func TestPackImportFetchesACodeAndQueuesItIntoANamedProfile(t *testing.T) {
	t.Setenv("MORTAR_ENABLE_GAMES", "lethal-company")
	const key = "019c9378-d2e6-4fec-5a2f-e4b22f8cf1d1"
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"export.r2x": "profileName: Just some mods\nmods:\n" +
			"  - {name: BepInEx-BepInExPack, version: {major: 5, minor: 4, patch: 2304}, enabled: true}\n" +
			"  - {name: AinaVT-LethalConfig, version: {major: 1, minor: 4, patch: 6}, enabled: false}\n",
		"config/ainavt.lc.lethalconfig.cfg": "[General]\n",
	} {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/experimental/legacyprofile/get/"+key+"/" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("#r2modman\n" + base64.StdEncoding.EncodeToString(buf.Bytes())))
	}))
	defer srv.Close()

	s := services(t)
	q := &recordQueue{}
	s.Packs = &packsvc.Service{Profiles: s.Store, Queue: q, Code: pack.Code{URL: srv.URL}}
	ctx := context.Background()

	pv, err := s.Handle(ctx, "pack.import", Params{Value: key, Preview: true})
	if got, ok := pv.(packsvc.Preview); err != nil || !ok || got.Name != "Just some mods" || len(got.Packages) != 2 || len(q.got) != 0 {
		t.Fatalf("preview = %+v, %v; queued %d", pv, err, len(q.got))
	}
	if _, err := s.Handle(ctx, "pack.import", Params{Value: key}); err == nil {
		t.Fatal("a code that names no game needs --game")
	}

	out, err := s.Handle(ctx, "pack.import", Params{Game: "lethal-company", Name: "Regress R2", Value: key})
	if err != nil {
		t.Fatal(err)
	}
	res, ok := out.(packsvc.Result)
	if !ok || res.Queued != 2 || len(q.got) != 2 || q.got[0].Profile != res.Profile || len(q.got[1].Disabled) != 1 {
		t.Fatalf("result %+v, queued %+v", res, q.got)
	}
	prof, err := s.resolve("lethal-company", "Regress R2")
	if err != nil || prof.ID != res.Profile {
		t.Fatalf("named profile %+v, %v", prof, err)
	}
	dir, err := s.Store.ProfileDir("lethal-company", prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "BepInEx", "config", "ainavt.lc.lethalconfig.cfg")); err != nil {
		t.Fatalf("the code's config file was not written: %v", err)
	}
}
