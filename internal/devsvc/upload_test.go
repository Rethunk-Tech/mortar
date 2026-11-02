package devsvc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
	"github.com/zalando/go-keyring"
)

func TestUploadRefusesAnUntrustedProject(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	p := svc.Settings.Get().GamePrefs("lethal-company").DevProjects[0]
	next := p
	next.Untrusted = true
	if err := svc.Settings.ReplaceDevProject("lethal-company", p, next); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UploadThunderstore(t.Context(), "lethal-company", dir); usererr.KindOf(err) != usererr.Unconfirmed {
		t.Errorf("err = %v", err)
	}
}

func TestUploadMapsASitesRefusalToAUserError(t *testing.T) {
	keyring.MockInit()
	if err := secret.Set("nexus", "k"); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		status int
		kind   usererr.Kind
		busy   bool
	}{
		{http.StatusUnauthorized, usererr.Permission, false},
		{http.StatusForbidden, usererr.Permission, false},
		{http.StatusTooManyRequests, usererr.RateLimited, true},
		{http.StatusBadGateway, usererr.Network, false},
		{http.StatusConflict, usererr.Invalid, false},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(c.status) }))
		svc, _ := testService(t, testTemplate)
		svc.HTTP, svc.NexusURL, svc.CurseForgeURL = srv.Client(), srv.URL, srv.URL
		dir := fixtureProject(t, svc)
		writePackedZip(t, dir)
		if err := writeSites(dir, siteIDs{Nexus: NexusIDs{FileID: "7"}, CurseForge: CurseForgeIDs{ProjectID: 3}}); err != nil {
			t.Fatal(err)
		}
		if err := svc.SetCurseForgeToken("t"); err != nil {
			t.Fatal(err)
		}
		_, nexusErr := svc.UploadNexus(t.Context(), "lethal-company", dir, "1.0.0")
		_, cfErr := svc.UploadCurseForge(t.Context(), "lethal-company", dir, "1.0.0")
		for site, err := range map[string]error{"nexus": nexusErr, "curseforge": cfErr} {
			if usererr.KindOf(err) != c.kind || errors.Is(err, source.ErrBusy) != c.busy {
				t.Errorf("%s %d: kind %v, err %v", site, c.status, usererr.KindOf(err), err)
			}
		}
		srv.Close()
	}
}

func TestChangelogOfTakesOneVersionsNotes(t *testing.T) {
	log := "# Changelog\n\n## 1.2.0\n\n- b\n- c\n\n## 1.1.0\n\n- a\n"
	for version, want := range map[string]string{"1.2.0": "- b\n- c", "1.1.0": "- a", "1.0.0": ""} {
		if got := changelogOf(log, version); got != want {
			t.Errorf("%s = %q", version, got)
		}
	}
}

func TestSiteIDsAreKeptInTheProjectsRecordAndOffered(t *testing.T) {
	keyring.MockInit()
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	configured := func() []string {
		sites, _ := svc.UploadSites(t.Context(), "lethal-company", dir)
		ids := []string{}
		for _, s := range sites {
			ids = append(ids, s.ID+fmt.Sprint(s.Configured))
		}
		return ids
	}
	if got := configured(); !slices.Equal(got, []string{"thunderstoretrue", "nexusfalse", "curseforgefalse"}) {
		t.Fatalf("sites before ids = %v", got)
	}
	info, err := svc.Inspect(t.Context(), "lethal-company", dir)
	if err != nil {
		t.Fatal(err)
	}
	edit := ProjectEdit{
		Name: info.Name, Description: info.Description, Author: info.Author, Team: info.Team, IDPrefix: info.IDPrefix,
		Categories: info.Categories, TestWith: info.TestWith,
		Nexus: NexusIDs{ModID: "55", FileID: "77"}, CurseForge: CurseForgeIDs{ProjectID: 31},
	}
	got, err := svc.Edit(t.Context(), "lethal-company", dir, edit)
	if err != nil || got.Nexus != edit.Nexus || got.CurseForge != edit.CurseForge {
		t.Fatalf("edited = %+v, %v", got, err)
	}
	body, _ := safefs.ReadFile(filepath.Join(dir, ".mortar", "template.json"))
	var rec map[string]any
	_ = json.Unmarshal(body, &rec)
	if fmt.Sprint(rec["nexus"], rec["curseforge"]) != "map[fileId:77 modId:55] map[projectId:31]" {
		t.Errorf("record = %s", body)
	}
	if got := configured(); !slices.Equal(got, []string{"thunderstoretrue", "nexustrue", "curseforgetrue"}) {
		t.Errorf("sites = %v", got)
	}
	edit.Nexus = NexusIDs{}
	edit.CurseForge = CurseForgeIDs{}
	if _, err := svc.Edit(t.Context(), "lethal-company", dir, edit); err != nil {
		t.Fatal(err)
	}
	body, _ = safefs.ReadFile(filepath.Join(dir, ".mortar", "template.json"))
	if strings.Contains(string(body), "nexus") || strings.Contains(string(body), "curseforge") {
		t.Errorf("cleared ids stay in %s", body)
	}
	edit.Nexus.FileID = "../x"
	if _, err := svc.Edit(t.Context(), "lethal-company", dir, edit); usererr.KindOf(err) != usererr.Invalid {
		t.Errorf("unsafe id: %v", err)
	}
}

func TestSiteIDsSurviveTheStartUpMigrationOfAProjectOfFiles(t *testing.T) {
	dir := t.TempDir()
	rec := `{"kind":"generic","name":"Pack","team":"Ab","nexus":{"modId":"5","fileId":"6"},"curseforge":{"projectId":7}}`
	if err := writeTracked(dir, assetRecordFile, []byte(rec)); err != nil {
		t.Fatal(err)
	}
	body, err := standardRecord(dir, "lethal-company")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeTracked(dir, assetRecordFile, body); err != nil {
		t.Fatal(err)
	}
	if got := readSites(dir); got.Nexus != (NexusIDs{ModID: "5", FileID: "6"}) || got.CurseForge.ProjectID != 7 {
		t.Errorf("ids after migration = %+v (%s)", got, body)
	}
}

func TestUploadClientRefusesARedirectToAnotherHost(t *testing.T) {
	var leaked string
	other := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { leaked = r.Header.Get("Apikey") }))
	defer other.Close()
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL, http.StatusTemporaryRedirect)
	}))
	defer first.Close()
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, first.URL, strings.NewReader("x"))
	req.Header.Set("Apikey", "secret")
	res, err := (&Service{}).httpClient().Do(req)
	if err == nil {
		_ = res.Body.Close()
		t.Fatal("followed a redirect to another host")
	}
	if leaked != "" {
		t.Errorf("the second host saw Apikey %q", leaked)
	}
}

func writePackedZip(t *testing.T, dir string) {
	t.Helper()
	release := filepath.Join(dir, "artifacts", "thunderstore", "release")
	if err := os.MkdirAll(release, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := safefs.WriteFile(filepath.Join(release, "Ab-ShopDrop-1.0.0.zip"), []byte("zip"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseNotesReadsTheVersionsSectionOfTheChangelog(t *testing.T) {
	withLog := t.TempDir()
	if err := safefs.WriteFile(filepath.Join(withLog, "CHANGELOG.md"), []byte("## 1.2.3\n\n- fix\n\n## 1.0.0\n\n- first\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		dir, version, want string
		invalid            bool
	}{
		{withLog, "1.0", "", true},
		{t.TempDir(), "1.2.3", "", false},
		{withLog, "1.2.3", "- fix", false},
	} {
		got, err := releaseNotes(c.dir, c.version)
		if got != c.want || (err != nil) != c.invalid || c.invalid && usererr.KindOf(err) != usererr.Invalid {
			t.Errorf("releaseNotes(%q) = %q, %v", c.version, got, err)
		}
	}
}

func TestUploadsNeedTheSitesIdsAndCredentials(t *testing.T) {
	keyring.MockInit()
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	ctx := t.Context()
	if _, err := svc.UploadThunderstore(ctx, "lethal-company", dir); usererr.KindOf(err) != usererr.NotFound || !strings.Contains(err.Error(), "pack") {
		t.Errorf("thunderstore without a package: %v", err)
	}
	writePackedZip(t, dir)
	if _, err := svc.UploadNexus(ctx, "lethal-company", dir, "1.0.0"); usererr.KindOf(err) != usererr.Invalid {
		t.Errorf("nexus without ids: %v", err)
	}
	if _, err := svc.UploadCurseForge(ctx, "lethal-company", dir, "1.0.0"); usererr.KindOf(err) != usererr.Invalid {
		t.Errorf("curseforge without ids: %v", err)
	}
	if err := writeSites(dir, siteIDs{CurseForge: CurseForgeIDs{ProjectID: 31}}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UploadCurseForge(ctx, "lethal-company", dir, "1.0.0"); usererr.KindOf(err) != usererr.Permission {
		t.Errorf("curseforge without a token: %v", err)
	}
}

func TestNewestPackagePicksTheLatestZip(t *testing.T) {
	dir := t.TempDir()
	if got := newestPackage(dir); got != "" {
		t.Fatalf("no packages = %q", got)
	}
	release := filepath.Join(dir, "artifacts", "thunderstore", "release")
	if err := os.MkdirAll(release, 0o750); err != nil {
		t.Fatal(err)
	}
	for name, at := range map[string]int64{"old.zip": 1_000_000, "new.zip": 2_000_000} {
		path := filepath.Join(release, name)
		if err := safefs.WriteFile(path, []byte("zip"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, time.Unix(at, 0), time.Unix(at, 0)); err != nil {
			t.Fatal(err)
		}
	}
	if got := newestPackage(dir); filepath.Base(got) != "new.zip" {
		t.Errorf("newest = %q", got)
	}
}
