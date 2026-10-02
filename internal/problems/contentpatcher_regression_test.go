package problems

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConflictWithBlankLoserIsCosmetic(t *testing.T) {
	t.Run("blank loser", func(t *testing.T) {
		winner := syntheticLoadPack(t, `{"Changes":[{"Action":"Load","Target":"Maps/Test","FromFile":"winner.json","Priority":"High"}]}`, map[string]string{
			"winner.json": `{"Tile": 1}`,
		})
		loser := syntheticLoadPack(t, `{"Changes":[{"Action":"Load","Target":"Maps/Test","FromFile":"LOSER.JSON","Priority":"Low"}]}`, map[string]string{
			"loser.JSON": `// harmless
[]`,
		})
		conflicts := assetConflicts([]Installed{winner, loser})
		if len(conflicts) != 1 || !conflicts[0].Cosmetic {
			t.Fatalf("expected a cosmetic load conflict, got %#v", conflicts)
		}
	})

	t.Run("blank winner still wipes data", func(t *testing.T) {
		winner := syntheticLoadPack(t, `{"Changes":[{"Action":"Load","Target":"Maps/Test","FromFile":"winner.json","Priority":"High"}]}`, map[string]string{
			"winner.json": `{}`,
		})
		loser := syntheticLoadPack(t, `{"Changes":[{"Action":"Load","Target":"Maps/Test","FromFile":"loser.json","Priority":"Low"}]}`, map[string]string{
			"loser.json": `{"Tile": 1}`,
		})
		conflicts := assetConflicts([]Installed{winner, loser})
		if len(conflicts) != 1 || conflicts[0].Cosmetic {
			t.Fatalf("expected a real load conflict, got %#v", conflicts)
		}
	})
}

func syntheticLoadPack(t *testing.T, content string, files map[string]string) Installed {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), []byte(`{"ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "content.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return Installed{Enabled: true, Folder: root, UniqueID: filepath.Base(root), Name: filepath.Base(root)}
}
