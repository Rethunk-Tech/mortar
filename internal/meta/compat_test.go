package meta

import (
	"strings"
	"testing"
)

// Canned sample of SmapiCompatibilityList data/mods.jsonc (and the wiki dump SMAPI.Web also served).
const compatSample = `[
  {
    "name": "Content Patcher",
    "id": "Pathoschild.ContentPatcher",
    "nexus": 1915,
    "compatibility": { "status": "ok" }
  },
  {
    "name": "Broken Example",
    "id": ["Author.Broken", "Author.Broken.Old"],
    "nexusID": 99,
    "compatibility": {
      "status": "Broken",
      "summary": "Crashes on load.",
      "brokeIn": "Stardew Valley 1.6"
    }
  },
  {
    "name": "Unofficial Example",
    "id": "Author.Unofficial",
    "nexus": 42,
    "compatibility": {
      "status": "unofficial",
      "summary": "Use the unofficial update.",
      "unofficialUrl": "https://example.com/unofficial"
    }
  },
  {
    "name": "Workaround Example",
    "id": "Author.Workaround",
    "compatibility": { "status": "workaround", "summary": "Load after X." }
  },
  {
    "name": "Obsolete Example",
    "id": "Author.Obsolete",
    "nexus": 7,
    "compatibility": { "status": "obsolete", "summary": "Use New Mod." },
    "successor": "Author.New"
  }
]
`

func TestParseCompatJSONMapsIDAndNexus(t *testing.T) {
	idx, err := parseCompatJSON([]byte(compatSample))
	if err != nil {
		t.Fatal(err)
	}
	ok, found := idx.Lookup("pathoschild.contentpatcher", 0)
	if !found || ok.Status != StatusOK {
		t.Fatalf("unique id: %+v found=%v", ok, found)
	}
	byNexus, found := idx.Lookup("", 1915)
	if !found || byNexus.Status != StatusOK {
		t.Fatalf("nexus: %+v found=%v", byNexus, found)
	}
	broken, found := idx.Lookup("Author.Broken.Old", 0)
	if !found || broken.Status != StatusBroken || broken.BrokeIn != "Stardew Valley 1.6" || broken.Summary != "Crashes on load." {
		t.Fatalf("alias id: %+v found=%v", broken, found)
	}
	if e, ok := idx.Lookup("", 99); !ok || e.Status != StatusBroken {
		t.Fatalf("broken nexus: %+v ok=%v", e, ok)
	}
	unoff, found := idx.Lookup("Author.Unofficial", 0)
	if !found || unoff.Status != StatusUnofficial || unoff.UnofficialURL != "https://example.com/unofficial" {
		t.Fatalf("unofficial: %+v found=%v", unoff, found)
	}
	opt, found := idx.Lookup("Author.Workaround", 0)
	if !found || opt.Status != StatusBroken {
		t.Fatalf("workaround→broken: %+v found=%v", opt, found)
	}
	obs, found := idx.Lookup("Author.Obsolete", 7)
	if !found || obs.Status != StatusObsolete || obs.Replacement != "Author.New" {
		t.Fatalf("obsolete: %+v found=%v", obs, found)
	}
}

func TestParseCompatJSONWikiObjectAndComments(t *testing.T) {
	body := `
// wiki-shaped dump
{
  "mods": [
    {
      "ID": ["Wiki.Mod"],
      "nexusID": 3,
      "compatibility": { "status": "Abandoned", "summary": "Gone." }
    },
  ]
}
`
	idx, err := parseCompatJSON([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	e, found := idx.Lookup("Wiki.Mod", 3)
	if !found || e.Status != StatusAbandoned || e.Summary != "Gone." {
		t.Fatalf("%+v found=%v err=%v", e, found, err)
	}
	if _, err := parseCompatJSON([]byte("not json")); err == nil {
		t.Fatal("want decode error")
	}
	if !strings.Contains(defaultCompatURL, "SmapiCompatibilityList") {
		t.Fatal(defaultCompatURL)
	}
}

func TestBrokenOnComparesTheGameVersion(t *testing.T) {
	cases := []struct {
		e    CompatEntry
		game string
		want bool
	}{
		{CompatEntry{Status: StatusBroken, BrokeIn: "Stardew Valley 1.6"}, "1.6.15", true},
		{CompatEntry{Status: StatusBroken, BrokeIn: "Stardew Valley 1.6.9"}, "1.6.8", false},
		{CompatEntry{Status: StatusBroken, BrokeIn: "Stardew Valley 1.7"}, "1.6.15", false},
		{CompatEntry{Status: StatusBroken, BrokeIn: "Stardew Valley 1.2?"}, "1.6.15", true},
		{CompatEntry{Status: StatusBroken, BrokeIn: "SMAPI 3.0"}, "1.6.15", true},
		{CompatEntry{Status: StatusBroken, BrokeIn: "never worked"}, "1.6.15", true},
		{CompatEntry{Status: StatusBroken, BrokeIn: "Stardew Valley 1.7"}, "", true},
		{CompatEntry{Status: StatusObsolete, BrokeIn: "Stardew Valley 1.6"}, "1.6.15", false},
	}
	for _, c := range cases {
		if got := c.e.BrokenOn(c.game); got != c.want {
			t.Errorf("%q on %q = %v, want %v", c.e.BrokeIn, c.game, got, c.want)
		}
	}
}

func TestCompatStatusDefaultsFollowTheSchema(t *testing.T) {
	idx, err := parseCompatJSON([]byte(`{"mods":[
		{"id":"A.Broke","nexus":1,"brokeIn":"Stardew Valley 1.6"},
		{"id":"A.Unofficial","brokeIn":"Stardew Valley 1.6","unofficialUpdate":{"url":"https://example.com"}},
		{"id":"A.Fine"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]string{"A.Broke": StatusBroken, "A.Unofficial": StatusUnofficial, "A.Fine": StatusOK} {
		if e, _ := idx.Lookup(id, 0); e.Status != want {
			t.Errorf("%s: %q, want %q", id, e.Status, want)
		}
	}
}

func TestCompatIndexTiesRefsToAUniqueIDUnlessAmbiguous(t *testing.T) {
	idx, err := parseCompatJSON([]byte(`[
	  {"id":"A.One","nexus":10,"github":"Me/One","status":"ok"},
	  {"id":"B.Two","github":"Me/Many","status":"ok"},
	  {"id":"B.Three","github":"me/many"},
	  {"id":["C.Four","C.Five"],"nexus":11}
	]`))
	if err != nil {
		t.Fatal(err)
	}
	if idx.NexusToID[10] != "A.One" || idx.GitHubToID["me/one"] != "A.One" {
		t.Fatalf("ties = %v %v", idx.NexusToID, idx.GitHubToID)
	}
	if _, ok := idx.GitHubToID["me/many"]; ok {
		t.Fatal("a repository of two mods names no identity")
	}
	if _, ok := idx.NexusToID[11]; ok {
		t.Fatal("a page of two mods names no identity")
	}
}

func TestParseCompatJSONKeepsUnofficialVersion(t *testing.T) {
	idx, err := parseCompatJSON([]byte(`[{"name":"Bus","id":"hootless.BusLocations","brokeIn":"Stardew Valley 1.6",
		"unofficialUpdate":{"version":"1.2.2-unofficial.1-Xytronix","url":"https://example.com/b"}}]`))
	if err != nil {
		t.Fatal(err)
	}
	e, ok := idx.Lookup("hootless.BusLocations", 0)
	if !ok || e.Status != StatusUnofficial || e.UnofficialVersion != "1.2.2-unofficial.1-Xytronix" || e.UnofficialURL != "https://example.com/b" {
		t.Fatalf("entry = %+v ok=%v", e, ok)
	}
}
