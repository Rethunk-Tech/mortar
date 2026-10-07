package contentpatcher

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/testenv/packs"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// bigPack writes a pack shaped like a large real one: an EditData with many object-valued entries, plus many
// Load and EditData changes with conditions, split over Included files.
func bigPack(t *testing.T, n int) framework.Mod {
	t.Helper()
	root := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("manifest.json", fmt.Sprintf(`{"UniqueID":"Big.Pack%d","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`, n))
	var entries []string
	for i := range 60 {
		entries = append(entries, fmt.Sprintf(`"item_%d_%d":{"Name":"Item %d","DisplayName":"%s","Description":"%s","Price":%d,"Texture":"Mods/Big/%d"}`,
			n, i, i, strings.Repeat("Display ", 6), strings.Repeat("A long description of the item. ", 6), 100+i, i))
	}
	var changes, includes []string
	for f := range 6 {
		changes = nil
		changes = append(changes, fmt.Sprintf(`{"Action":"EditData","Target":"Data/Objects%d","Entries":{%s}}`, f, strings.Join(entries, ",")))
		for i := range 40 {
			changes = append(changes, fmt.Sprintf(`{"Action":"Load","Target":"Characters/Big%d_%d_%d","FromFile":"assets/{{TargetWithoutPath}}.png","When":{"Season":"spring, summer","HasMod":"Some.Mod%d"}}`, n, f, i, i%5))
		}
		write(fmt.Sprintf("part%d.json", f), `{"Changes":[`+strings.Join(changes, ",")+`]}`)
		includes = append(includes, fmt.Sprintf(`{"Action":"Include","FromFile":"part%d.json"}`, f))
	}
	write("content.json", `{"Changes":[`+strings.Join(includes, ",")+`]}`)
	return packs.FromDisk(framework.Mod{Enabled: true, Folder: root, UniqueID: fmt.Sprintf("Big.Pack%d", n), Name: "Big", Key: fmt.Sprintf("Big.Pack%d", n)})
}

func liveHeap() uint64 {
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapAlloc
}

// retainedPerPackBudget is the heap one parsed pack of bigPack's shape may hold while the cache keeps it. It was
// 608,000 bytes when every data entry's whole JSON text rode in its shape; digests, shared strings and trimmed
// slices bring it to about 385,000.
const retainedPerPackBudget = 420 << 10

func TestParsedPackRetainedBytes(t *testing.T) {
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)
	const count = 20
	mods := make([]framework.Mod, count)
	for i := range mods {
		mods[i] = bigPack(t, i)
	}
	before := liveHeap()
	held := make([]cachedPack, count)
	for i, im := range mods {
		held[i] = readContentPack(im)
	}
	after := liveHeap()
	per := (after - before) / count
	t.Logf("retained per parsed pack: %d bytes (%d patches)", per, len(held[0].patches))
	runtime.KeepAlive(held)
	if per > retainedPerPackBudget {
		t.Fatalf("a parsed pack holds %d bytes, budget %d", per, retainedPerPackBudget)
	}
}

func TestCompactLiteralKeepsEqualityAndDropsBodies(t *testing.T) {
	long := `{"Name":"Item","Description":"` + strings.Repeat("x", 500) + `"}`
	a, b := compactLiteral(long), compactLiteral(strings.ToUpper(long))
	if a != b || len(a) > 40 {
		t.Fatalf("equal text must digest alike and small: %q vs %q", a, b)
	}
	if compactLiteral(long) == compactLiteral(long+" ") || compactLiteral("5") != "5" {
		t.Fatal("different text must differ and short values stay readable")
	}
	same := cpShape{kind: 'p', key: "entry:k", value: a}
	if same.overlaps(cpShape{kind: 'p', key: "entry:k", value: b}) {
		t.Fatal("two packs writing the same entry body agree")
	}
	if !same.overlaps(cpShape{kind: 'p', key: "entry:k", value: compactLiteral(long + "y")}) {
		t.Fatal("different entry bodies conflict")
	}
}
