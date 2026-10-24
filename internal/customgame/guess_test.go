package customgame

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGuessFromReadsTheGamesFiles(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files []string
		exe   string
		want  Guess
	}{
		{"unreal", nil, "Void/VotV/Binaries/Win64/VotV-Win64-Shipping.exe", Guess{"Void", Shimloader}},
		{"il2cpp", []string{"Game/GameAssembly.dll"}, "Game/Game.exe", Guess{"Game", BepInEx6}},
		{"mono", []string{"Game/Game_Data/Managed/Assembly-CSharp.dll"}, "Game/Game.exe", Guess{"Game", BepInEx5}},
		{"melonloader wins over il2cpp", []string{"Game/GameAssembly.dll", "Game/MelonLoader/x.dll"}, "Game/Game.exe", Guess{"Game", MelonLoader}},
		{"unknown", nil, "Some Game/run.exe", Guess{"Some Game", ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for _, f := range tc.files {
				p := filepath.Join(root, f)
				if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, nil, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if got := GuessFrom(filepath.Join(root, tc.exe)); got != tc.want {
				t.Fatalf("GuessFrom = %+v, want %+v", got, tc.want)
			}
		})
	}
}
