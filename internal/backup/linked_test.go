package backup

import (
	"database/sql"
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/saves"

	_ "modernc.org/sqlite"
)

// A Vintage Story world's minimap, Maps/<SavegameIdentifier>.db beside the saves folder, is backed up and restored
// with the world.
func TestBackupCarriesAndRestoresAWorldsMinimap(t *testing.T) {
	data := t.TempDir()
	dir := filepath.Join(data, "Saves")
	if err := os.MkdirAll(filepath.Join(data, "Maps"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	const id = "8c4f2a10-1b2c-4d5e-9f00-123456789abc"
	game := binary.AppendUvarint(nil, 34<<3|2)
	game = binary.AppendUvarint(game, uint64(len(id)))
	game = append(game, id...)
	conn, err := sql.Open("sqlite", filepath.Join(dir, "w.vcdbs"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(t.Context(), "CREATE TABLE gamedata (savegameid integer PRIMARY KEY, data BLOB)"); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(t.Context(), "INSERT INTO gamedata VALUES (1, ?)", game); err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	mapFile := filepath.Join(data, "Maps", id+".db")
	if err := safefs.WriteFile(mapFile, []byte("map then"), 0o600); err != nil {
		t.Fatal(err)
	}
	l := saves.Layout{Dir: dir, Files: []string{"*.vcdbs"}, Linked: []string{"Maps/{uuid}.db"}}
	out := t.TempDir()
	start := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	whole, err := Saves(t.Context(), l, out, DefaultKeep, start, Cause{Kind: KindManual})
	if err != nil {
		t.Fatal(err)
	}
	if got := zipNames(t, whole); !slices.Contains(got, "Maps/"+id+".db") {
		t.Fatalf("zip = %v, want the minimap beside Saves/", got)
	}
	if err := safefs.WriteFile(mapFile, []byte("map now"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Restore(t.Context(), whole, l, out, []string{"w.vcdbs"}, DefaultKeep, start.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if b, err := safefs.ReadFile(mapFile); err != nil || string(b) != "map then" {
		t.Fatalf("minimap after restore = %q, %v", b, err)
	}
}
