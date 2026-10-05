package bepinex5

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestAnalyze(t *testing.T) {
	// The first log is a trimmed public LogOutput.log (Sanunez/LethalPack); the rest use the format strings of BepInEx 5.4.x's
	// BaseChainloader and AssemblyPatcher.
	sample, err := fsx.ReadFile("testdata/logoutput-missing-dependency.log")
	if err != nil {
		t.Fatal(err)
	}
	player := "NullReferenceException: Object reference not set to an instance of an object\n" +
		"  at UnityEngine.Object.Instantiate (UnityEngine.Object original) [0x00000] in <aa>:0 \n" +
		"  at LateCompany.Patches.ConnectionApproval_Patch.Postfix (Unity.Netcode.NetworkManager request) [0x00019] in <9c56>:0 \n" +
		"  at (wrapper dynamic-method) GameNetworkManager.DMD<GameNetworkManager::ConnectionApproval>(GameNetworkManager)\n" +
		"\nNullReferenceException: Object reference not set to an instance of an object\n" +
		"  at LateCompany.Patches.ConnectionApproval_Patch.Postfix (Unity.Netcode.NetworkManager request) [0x00019] in <9c56>:0 \n" +
		"ArgumentException: only game code\n  at GameNetworkManager.Start () [0x00000] in <bb>:0 \n"
	cases := []struct {
		name, log, player string
		want              []Finding
	}{
		{"real log", string(sample), "", []Finding{
			{KindMissingDependency, "Youtube Boombox 1.5.0", "missing dependencies: LC_API", 4, "LogOutput.log"},
			{KindPatchException, "LateCompany", "System.NullReferenceException: Object reference not set to an instance of an object", 5, "LogOutput.log"},
		}},
		{"newer message wording", "[Error  :   BepInEx] Could not load [A 1.0.0] because it has missing dependencies: B (>=2.0.0). Install the listed plugin(s) and restart the game.\n", "", []Finding{
			{KindMissingDependency, "A 1.0.0", "missing dependencies: B (>=2.0.0)", 1, "LogOutput.log"},
		}},
		{"incompatible", "[Error  :   BepInEx] Could not load [A 1.0.0] because it has incompatible dependencies: B (found 1.0.0, requires 2.0.0)\n" +
			"[Error  :   BepInEx] Could not load [C 1.0.0] because the following dependencies are installed with an incompatible version: BepInEx (found 5.4.21, requires >=5.4.22). Update the listed plugin(s) to a version that satisfies the requirement.\n", "", []Finding{
			{KindIncompatible, "A 1.0.0", "incompatible dependencies: B (found 1.0.0, requires 2.0.0)", 1, "LogOutput.log"},
			{KindIncompatible, "C 1.0.0", "incompatible dependencies: BepInEx (found 5.4.21, requires >=5.4.22)", 2, "LogOutput.log"},
		}},
		{"load exception and preloader", "[Info   :   BepInEx] Loading [A 1.0.0]\r\n[Error  :   BepInEx] Error loading [A 1.0.0]: System.TypeLoadException: no such type\r\n  at X\r\n" +
			"[Error  : Preloader] Failed to run [PatcherA.Patcher] when patching [Assembly-CSharp]. This assembly will not be patched. Error: boom\n" +
			"[Fatal  :   BepInEx] Could not run preloader!\n", "", []Finding{
			{KindLoadException, "A 1.0.0", "System.TypeLoadException: no such type", 2, "LogOutput.log"},
			{KindPreloader, "", "Failed to run [PatcherA.Patcher] when patching [Assembly-CSharp]. This assembly will not be patched. Error: boom", 4, "LogOutput.log"},
			{KindPreloader, "", "Could not run preloader!", 5, "LogOutput.log"},
		}},
		{"player log blames the first mod frame once", "", player, []Finding{
			{KindUnityException, "LateCompany", "NullReferenceException: Object reference not set to an instance of an object", 1, "Player.log"},
		}},
		{"nothing wrong", "[Info   :   BepInEx] Chainloader started\n", "Loading player data\n", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Analyze(c.log, c.player)
			if len(got) != len(c.want) {
				t.Fatalf("got %+v, want %+v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("finding %d = %+v, want %+v", i, got[i], c.want[i])
				}
			}
		})
	}
}
