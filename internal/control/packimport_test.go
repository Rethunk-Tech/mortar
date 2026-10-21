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
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/pack"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/sharesvc"
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
	s.Shares = sharesvc.NewService(sharesvc.Deps{
		Profiles: s.Store, Queue: q, PackCode: pack.Code{URL: srv.URL},
		SignedIn: func() bool { return true }, Premium: func() bool { return false },
		Env: func(string) problems.Environment { return problems.Environment{} },
	})
	ctx := context.Background()

	pv, err := s.Handle(ctx, "pack.import", Params{Game: "lethal-company", Value: key, Preview: true})
	if got, ok := pv.(sharesvc.Preview); err != nil || !ok || got.Name != "Just some mods" || len(got.Mods) < 2 || len(q.got) != 0 {
		t.Fatalf("preview = %+v, %v; queued %d", pv, err, len(q.got))
	}
	if _, err := s.Handle(ctx, "pack.import", Params{Value: key, Preview: true}); err == nil {
		t.Fatal("a code that names no game needs --game")
	}

	out, err := s.Handle(ctx, "pack.import", Params{Game: "lethal-company", Name: "Regress R2", Value: key})
	if err != nil {
		t.Fatal(err)
	}
	res, ok := out.(sharesvc.Result)
	if !ok || res.Queued != len(q.got) || len(q.got) < 2 || q.got[0].Profile != res.Profile.ID || !slices.ContainsFunc(q.got, func(r queue.Request) bool { return len(r.Disabled) == 1 }) {
		t.Fatalf("result %+v, queued %+v", res, q.got)
	}
	prof, err := s.resolve("lethal-company", "Regress R2")
	if err != nil || prof.ID != res.Profile.ID {
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
