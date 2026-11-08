package contentpatcher

import (
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework/contentpatcher/cp"
	"github.com/Rethunk-Tech/mortar/internal/testenv/packs"
)

// A data key's label is Mortar's wording, without the pack scoping a token key carries.
func TestKeyLabelDropsPackScope(t *testing.T) {
	if got := keyLabel("field:301/ContextTags/@/mods/A|{{ModId}}_x.Price"); got != "Field 301/ContextTags/{{ModId}}_x.Price" {
		t.Fatalf("label %q", got)
	}
}

// The evidence of a data conflict is worded by Mortar: its entries are "Entry 123" and its conditions in words.
func TestConflictEvidenceEditData(t *testing.T) {
	cp.SetCacheRoot(t.TempDir())
	t.Cleanup(func() { cp.SetCacheRoot("") })
	pack := func(id, value string) cp.Mod {
		content := `{"Changes":[{"Action":"EditData","Target":"Data/Objects","Priority":"Late","When":{"Season":"Spring"},"Entries":{"123":"` + value + `"}}]}`
		return cpMod(packs.Disk(t, id, map[string]string{"manifest.json": packs.Manifest(id), "content.json": content}))
	}
	mods := []cp.Mod{pack("Pack.DataA", "alpha"), pack("Pack.DataB", "beta")}
	got := toFrameworkFindings(cp.Analyze(cp.Request{Enabled: mods, All: mods, Order: smapiLoadOrder}))
	if len(got.AssetConflicts) != 1 || got.AssetConflicts[0].Kind != "edit" || got.AssetConflicts[0].Target != "data/objects" {
		t.Fatalf("got %+v", got.AssetConflicts)
	}
	ev := got.AssetConflicts[0].Evidence
	if len(ev) != 2 {
		t.Fatalf("evidence = %+v", ev)
	}
	for _, e := range ev {
		if len(e.Keys) != 1 || e.Keys[0] != "Entry 123" {
			t.Fatalf("keys %q", e.Keys)
		}
		if e.Action != "EditData" || e.Source != "content.json" || e.Index != 0 || e.When == "" || e.Priority != "Late" {
			t.Fatalf("evidence %+v", e)
		}
		if !strings.Contains(e.When, "season=spring") {
			t.Fatalf("when %q", e.When)
		}
	}
}

// whenSummaryOf words the conditions of a pack whose one change carries the given content.json.
func whenSummaryOf(t *testing.T, content string) string {
	t.Helper()
	pack := packs.Disk(t, "Me.When", map[string]string{"manifest.json": packs.Manifest("Me.When"), "content.json": content})
	for _, patch := range cp.ReadPack(cpMod(pack), false).Patches {
		if patch.Action != "" {
			return whenSummary(patch.When)
		}
	}
	t.Fatalf("no change read from %s", content)
	return ""
}

func TestWhenSummaryPlainDynamicValue(t *testing.T) {
	content := `{"DynamicTokens":[{"Name":"LivingWithSen","Value":"true"}],"Changes":[
		{"Action":"EditData","Target":"Data/Objects","Entries":{"123":"x"},"When":{"LivingWithSen":"true"}}]}`
	if got := whenSummaryOf(t, content); got != "livingwithsen=true" {
		t.Fatalf("summary %q", got)
	}
}

func TestWhenSummaryNamesAssumedConditions(t *testing.T) {
	content := `{"Changes":[{"Action":"EditData","Target":"Data/Objects","Entries":{"123":"x"},"When":{
		"HasSeenEvent": "12345",
		"HasFlag: hostPlayer": "beenToDesert",
		"Hearts:Abigail": "8",
		"Time": "{{Range: 0600, 1200}}",
		"HasProfession |contains=Gemologist": false,
		"Season": "Spring"}}]}`
	want := "HasFlag beentodesert; season=spring; also needs: HasSeenEvent, Hearts:Abigail, Time, not HasProfession |contains=Gemologist"
	if got := whenSummaryOf(t, content); got != want {
		t.Fatalf("summary %q, want %q", got, want)
	}
}

// The conflict names the cells its packs share, and Mortar words them as pixel positions.
func TestItemConflictInfo(t *testing.T) {
	if got := itemConflictInfo([]string{"13,12"}); got != "cell (208, 192)" {
		t.Fatalf("item info = %q", got)
	}
	if got := itemConflictInfo(nil); got != "" {
		t.Fatalf("no cells = %q", got)
	}
}
