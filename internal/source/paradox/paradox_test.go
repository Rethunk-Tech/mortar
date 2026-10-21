package paradox

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := fsx.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// fake replays the recorded answers; a details request for modId 1 answers with another game's mod, 2 is refused and
// 3 names a file that would climb out of the content address.
func fake(t *testing.T, searched *map[string]any) Driver {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/mods":
			if searched != nil {
				_ = json.NewDecoder(r.Body).Decode(searched)
			}
			_, _ = w.Write(fixture(t, "search.json"))
		case r.URL.Path == "/mods/versions":
			_, _ = w.Write(fixture(t, "versions.json"))
		case r.URL.Path == "/mods" && r.URL.Query().Get("os") != "Windows":
			http.Error(w, "details are read for Windows", http.StatusBadRequest)
		case r.URL.Query().Get("modId") == "1":
			_, _ = w.Write(fixture(t, "other_game.json"))
		case r.URL.Query().Get("modId") == "2":
			_, _ = w.Write(fixture(t, "refused.json"))
		case r.URL.Query().Get("modId") == "3":
			_, _ = w.Write([]byte(`{"result":"OK","modDetail":{"game":"cities_skylines_2","name":"../../x","modId":"3"}}`))
		default:
			_, _ = w.Write(fixture(t, "detail.json"))
		}
	}))
	t.Cleanup(srv.Close)
	return Driver{HTTP: srv.Client(), URL: srv.URL}
}

func TestSearchDecodesTheListingAndSendsTheQuery(t *testing.T) {
	var sent map[string]any
	page, err := fake(t, &sent).Search(t.Context(), source.Query{
		Key: "cities_skylines_2", Page: 2, Size: 50, Text: "road", Sort: source.SortName,
		Categories: []string{"Map"}, ExcludeCategories: []string{"north american"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if sent["gameName"] != "cities_skylines_2" || sent["page"] != float64(2) || sent["pageSize"] != float64(50) ||
		sent["searchQuery"] != "road" || sent["sortBy"] != "displayName" || sent["orderBy"] != "asc" || fmt.Sprint(sent["tags"]) != "[Map]" {
		t.Fatalf("request body = %v", sent)
	}
	// The second mod is excluded by its tag and the third has an id that is no number.
	if page.Total != 27068 || page.Hidden != 1 || len(page.Items) != 1 {
		t.Fatalf("page = %+v", page)
	}
	it := page.Items[0]
	if it.ID != "74604" || it.Name != "Anarchy" || it.Downloads != 1047436 || it.Endorsements != 14464 || it.Category != "Code Mod" ||
		it.URL != "https://mods.paradoxplaza.com/mods/74604/Windows" || !strings.HasPrefix(it.Picture, "https://modscontent.") {
		t.Fatalf("item = %+v", it)
	}
}

func TestSearchOrdersByBestWhenNothingIsAsked(t *testing.T) {
	var sent map[string]any
	if _, err := fake(t, &sent).Search(t.Context(), source.Query{Key: "cities_skylines_2"}); err != nil {
		t.Fatal(err)
	}
	if sent["sortBy"] != "best" || sent["searchQuery"] != nil || sent["pageSize"] != float64(source.PageSize) {
		t.Fatalf("request body = %v", sent)
	}
	if _, err := fake(t, nil).Search(t.Context(), source.Query{Key: "cities_skylines"}); err == nil {
		t.Fatal("a game the driver does not serve searched")
	}
}

func TestResolveBuildsTheDownloadFromTheNewestVersionOrANamedOne(t *testing.T) {
	d := fake(t, nil)
	for _, tc := range []struct {
		wanted, version string
		id              int
	}{{"", "1.8.0", 40}, {"39", "1.7.10", 39}, {"1.7.10", "1.7.10", 39}, {"38", "38", 38}} {
		r, err := d.Resolve(t.Context(), "74604", tc.wanted)
		if err != nil || r.Version != tc.version || r.VersionID != tc.id {
			t.Fatalf("Resolve(%q) = %+v, %v", tc.wanted, r, err)
		}
	}
	r, _ := d.Resolve(t.Context(), "74604", "")
	const name = "25faeda9-b17e-4525-a08f-ba219ed13524"
	if r.URL != contentURL+"/cities_skylines_2/"+name+"/content/sources/"+name+"_40.zip" || r.FileName != name+"_40.zip" ||
		r.Size != 670237 || r.Name != "Anarchy" {
		t.Fatalf("resolved = %+v", r)
	}
	// The DLC and the mod's own entry are not mods to fetch.
	if !slices.Equal(r.Dependencies, []string{"74417"}) {
		t.Fatalf("dependencies = %v", r.Dependencies)
	}
	if _, err := d.Resolve(t.Context(), "74604", "9.9"); err == nil {
		t.Fatal("a version the mod lacks resolved")
	}
}

func TestRefusedAnswersAreErrors(t *testing.T) {
	d := fake(t, nil)
	for _, id := range []string{"1", "2", "3", "../x", ""} {
		if r, err := d.Resolve(t.Context(), id, ""); err == nil {
			t.Fatalf("Resolve(%q) = %+v", id, r)
		}
	}
	if _, err := d.Details(t.Context(), "", "2", ""); err == nil || !strings.Contains(err.Error(), "operating system") {
		t.Fatalf("a refusal sent as a 200 = %v", err)
	}
}

func TestItemsAnswerWithTheNewestVersionLabel(t *testing.T) {
	got, err := fake(t, nil).Items(t.Context(), "", "", []string{"74604", "../x"})
	if err != nil || len(got) != 1 || got["74604"].Version != "1.8.0" || got["74604"].Size != 670237 {
		t.Fatalf("items = %+v, %v", got, err)
	}
}

func TestVersionsWithoutAnAuthorNameAreLabelledByTheirNumber(t *testing.T) {
	d := fake(t, nil)
	det, err := d.Details(t.Context(), "", "74604", "")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(det.Versions, []string{"1.8.0", "1.7.10", "38", "1.4.0"}) || !slices.Equal(det.Dependencies, []string{"74417"}) ||
		!slices.Equal(det.Categories, []string{"Code Mod"}) || det.Changelog != "Patch **1.8.0**" || !strings.Contains(det.Description, "## Dependencies") {
		t.Fatalf("details = %+v", det)
	}
}

func TestChangelogStopsAtTheInstalledVersion(t *testing.T) {
	got, err := fake(t, nil).Changelog(t.Context(), "", "74604", "1.7.9", "1.8.0", "")
	if err != nil || len(got) != 2 || got[0].Version != "1.8.0" || got[0].Date != "2026-09-15" || got[1].Version != "1.7.10" {
		t.Fatalf("notes = %+v, %v", got, err)
	}
}
