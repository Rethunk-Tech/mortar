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
		{"unreal", nil, "Void/VotV/Binaries/Win64/VotV-Win64-Shipping.exe", Guess{Name: "Void", Loader: Shimloader}},
		{"il2cpp", []string{"Game/GameAssembly.dll"}, "Game/Game.exe", Guess{Name: "Game", Loader: BepInEx6}},
		{"mono", []string{"Game/Game_Data/Managed/Assembly-CSharp.dll"}, "Game/Game.exe", Guess{Name: "Game", Loader: BepInEx5}},
		{"melonloader wins over il2cpp", []string{"Game/GameAssembly.dll", "Game/MelonLoader/x.dll"}, "Game/Game.exe", Guess{Name: "Game", Loader: MelonLoader}},
		{"unknown", nil, "Some Game/run.exe", Guess{Name: "Some Game"}},
		{"gog graveyard keeper", []string{
			"Graveyard Keeper/Graveyard Keeper_Data/Managed/Assembly-CSharp.dll",
			"Graveyard Keeper/goggame-1780408621.info",
		}, "Graveyard Keeper/Graveyard Keeper.exe", Guess{Name: "Graveyard Keeper", Loader: BepInEx5, Warning: WarningGOG32Bit}},
		{"steam graveyard keeper", []string{"Graveyard Keeper/Graveyard Keeper_Data/Managed/Assembly-CSharp.dll"}, "Graveyard Keeper/Graveyard Keeper.exe", Guess{Name: "Graveyard Keeper", Loader: BepInEx5}},
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
