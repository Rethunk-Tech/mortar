package saves

import (
	"database/sql"
	"encoding/binary"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func protoString(b []byte, num uint64, s string) []byte {
	b = binary.AppendUvarint(b, num<<3|2)
	b = binary.AppendUvarint(b, uint64(len(s)))
	return append(b, s...)
}

func TestScanReadsAVintageStoryWorld(t *testing.T) {
	dir := t.TempDir()
	var data []byte
	data = binary.AppendUvarint(data, 7<<3|5) // a fixed32 the reader skips
	data = append(data, 1, 2, 3, 4)
	data = protoString(data, vsWorldName, "Bjornheim")
	data = binary.AppendUvarint(data, 40<<3|3) // a group whose own name field must not win
	data = protoString(data, vsWorldName, "inner")
	data = binary.AppendUvarint(data, 40<<3|4)
	data = binary.AppendUvarint(data, vsTotalSecondsPlayed<<3)
	data = binary.AppendUvarint(data, 7200)
	data = protoString(data, vsLastSavedVersion, "1.21.4")
	data = protoString(data, vsPlayStyle, "surviveandbuild")
	// ModData["BlockIDs"]: a map of block ids to codes, a mod's codes in its own domain.
	var ids []byte
	for i, code := range []string{"air", "creative:lamp", "primitivesurvival:fishbasket", "PrimitiveSurvival:weir"} {
		var pair []byte
		pair = binary.AppendUvarint(pair, 1<<3)
		pair = binary.AppendUvarint(pair, uint64(i))
		pair = protoString(pair, 2, code)
		ids = binary.AppendUvarint(ids, 1<<3|2)
		ids = binary.AppendUvarint(ids, uint64(len(pair)))
		ids = append(ids, pair...)
	}
	entry := protoString(nil, 1, "BlockIDs")
	entry = binary.AppendUvarint(entry, 2<<3|2)
	entry = binary.AppendUvarint(entry, uint64(len(ids)))
	entry = append(entry, ids...)
	data = binary.AppendUvarint(data, vsModData<<3|2)
	data = binary.AppendUvarint(data, uint64(len(entry)))
	data = append(data, entry...)
	conn, err := sql.Open("sqlite", filepath.Join(dir, "Bjornheim.vcdbs"))
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"PRAGMA journal_mode=WAL", "CREATE TABLE gamedata (savegameid integer PRIMARY KEY, data BLOB)"} {
		if _, err := conn.ExecContext(t.Context(), q); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := conn.ExecContext(t.Context(), "INSERT INTO gamedata VALUES (1, ?)", data); err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	if err := safefs.WriteFile(filepath.Join(dir, "Broken.vcdbs"), []byte("not sqlite"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &Scanner{Dir: dir, Files: []string{"*.vcdbs"}, Companions: []string{".vcdbs-shm", ".vcdbs-wal"}, CacheDir: t.TempDir()}
	infos, err := s.Scan(t.Context(), nil)
	if err != nil || len(infos) != 2 {
		t.Fatalf("Scan = %+v, %v", infos, err)
	}
	for _, in := range infos {
		switch in.Folder {
		case "Bjornheim.vcdbs":
			if in.Name != "Bjornheim" || in.GameVersion != "1.21.4" || in.PlayStyle != "surviveandbuild" || in.MillisecondsPlayed != 7_200_000 || !in.Unrecorded ||
				!slices.Equal(in.Used, []mod.ID{"modinfo:primitivesurvival"}) {
				t.Errorf("world = %+v", in)
			}
		case "Broken.vcdbs":
			if in.Name != "" || in.Played == 0 {
				t.Errorf("an unreadable world should keep only its file details: %+v", in)
			}
		}
	}
}
