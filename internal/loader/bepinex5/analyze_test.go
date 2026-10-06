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
	// Lines of a sandbox regress run of Lethal Company with Mask Fixes and Host Fixes.
	regress, err := fsx.ReadFile("testdata/logoutput-regress.log")
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
	awake := "[Info   :   BepInEx] Loading [Matrix throw 1.0.0]\n" +
		"[Error  : Unity Log] InvalidOperationException: matrix probe failed in Awake on purpose\n" +
		"Stack trace:\n" +
		"MortarMatrix.MatrixProbe.Awake () (at <6f3c2a1e5b7d4c8a9e0f1b2c3d4e5f60>:0)\n" +
		"UnityEngine.GameObject:AddComponent(Type)\n" +
		"BepInEx.Bootstrap.Chainloader:Start()\n" +
		"[Error  : Unity Log] NullReferenceException: Object reference not set to an instance of an object\n" +
		"Stack trace:\n" +
		"GameNetcodeStuff.PlayerControllerB.Update () (at <af9b1eec498a45aebd42601d6ab85015>:IL_0B14)\n" +
		"[Info   :   BepInEx] Chainloader startup complete\n"
	cases := []struct {
		name, log, player string
		want              []Finding
	}{
		{"real log", string(sample), "", []Finding{
			dep(fnd(KindMissingDependency, "Youtube Boombox 1.5.0", "missing dependencies: LC_API", 4, "LogOutput.log"), "LC_API"),
			fnd(KindPatchException, "LateCompany", "System.NullReferenceException: Object reference not set to an instance of an object", 5, "LogOutput.log"),
		}},
		{"newer message wording", "[Error  :   BepInEx] Could not load [A 1.0.0] because it has missing dependencies: B (>=2.0.0). Install the listed plugin(s) and restart the game.\n", "", []Finding{
			dep(fnd(KindMissingDependency, "A 1.0.0", "missing dependencies: B (>=2.0.0)", 1, "LogOutput.log"), "B"),
		}},
		{"incompatible", "[Error  :   BepInEx] Could not load [A 1.0.0] because it has incompatible dependencies: B (found 1.0.0, requires 2.0.0)\n" +
			"[Error  :   BepInEx] Could not load [C 1.0.0] because the following dependencies are installed with an incompatible version: BepInEx (found 5.4.21, requires >=5.4.22). Update the listed plugin(s) to a version that satisfies the requirement.\n", "", []Finding{
			fnd(KindIncompatible, "A 1.0.0", "incompatible dependencies: B (found 1.0.0, requires 2.0.0)", 1, "LogOutput.log"),
			fnd(KindIncompatible, "C 1.0.0", "incompatible dependencies: BepInEx (found 5.4.21, requires >=5.4.22)", 2, "LogOutput.log"),
		}},
		{"load exception and preloader", "[Info   :   BepInEx] Loading [A 1.0.0]\r\n[Error  :   BepInEx] Error loading [A 1.0.0]: System.TypeLoadException: no such type\r\n  at X\r\n" +
			"[Error  : Preloader] Failed to run [PatcherA.Patcher] when patching [Assembly-CSharp]. This assembly will not be patched. Error: boom\n" +
			"[Fatal  :   BepInEx] Could not run preloader!\n", "", []Finding{
			fnd(KindLoadException, "A 1.0.0", "System.TypeLoadException: no such type", 2, "LogOutput.log"),
			fnd(KindPreloader, "", "Failed to run [PatcherA.Patcher] when patching [Assembly-CSharp]. This assembly will not be patched. Error: boom", 4, "LogOutput.log"),
			fnd(KindPreloader, "", "Could not run preloader!", 5, "LogOutput.log"),
		}},
		{"player log blames the first mod frame once", "", player, []Finding{
			fnd(KindUnityException, "LateCompany", "NullReferenceException: Object reference not set to an instance of an object", 1, "Player.log"),
		}},
		{"regress run", string(regress), "", []Finding{
			fnd(KindPatchException, "Mask Fixes", "Mimic kill transpiler failed", 2, "LogOutput.log"),
			fnd(KindPatchException, "HostFixes", "ArgumentException: Undefined target method for patch method static System.Collections.Generic.IEnumerable<HarmonyLib.CodeInstruction> HostFixes.Plugin+ServerRPCMessageHandlers+UpdatePlayerPositionServerRpc_Transpile::UseServerRpcParams(System.Collections.Generic.IEnumerable<HarmonyLib.CodeInstruction> instructions)", 6, "LogOutput.log"),
		}},
		{"chainloader messages", "[Warning:   BepInEx] Skipping [LateCompany 1.0.10] because a newer version exists (LateCompany 1.0.12)\n" +
			"[Error  :   BepInEx] Could not load [MoreCompany 1.9.1] because it is incompatible with: me.swipez.melonloader.morecompany.lite\n" +
			"[Warning:   BepInEx] Plugin [LethalLib 0.16.0] targets a wrong version of BepInEx (5.4.22.0) and might not work until you update\n" +
			"[Warning:   BepInEx] Skipping [Youtube Boombox 1.5.0] because it has a dependency that was not loaded. See previous errors for details.\n" +
			"[Error  :   BepInEx] Error loading [A 1.0.0] : Could not load type 'X' from assembly 'Y'.\n" +
			"[Fatal  :   BepInEx] Error occurred starting the game\n" +
			"[Error  :Brutal Company] Event table is empty\n[Error  :Brutal Company] Event table is still empty\n" +
			"[Warning:   BepInEx] Skipping [Desktop Only 1.0.0] because of process filters (Lethal Company Server)\n", "", []Finding{
			fnd(KindDuplicateGUID, "LateCompany 1.0.10", "a newer copy loaded instead: LateCompany 1.0.12", 1, "LogOutput.log"),
			fnd(KindIncompatiblePlugin, "MoreCompany 1.9.1", "incompatible with: me.swipez.melonloader.morecompany.lite", 2, "LogOutput.log"),
			fnd(KindLoaderVersion, "LethalLib 0.16.0", "Plugin [LethalLib 0.16.0] targets a wrong version of BepInEx (5.4.22.0) and might not work until you update", 3, "LogOutput.log"),
			fnd(KindDependencyNotLoaded, "Youtube Boombox 1.5.0", "Skipping [Youtube Boombox 1.5.0] because it has a dependency that was not loaded. See previous errors for details.", 4, "LogOutput.log"),
			fnd(KindLoadException, "A 1.0.0", "Could not load type 'X' from assembly 'Y'.", 5, "LogOutput.log"),
			fnd(KindChainloader, "", "Error occurred starting the game", 6, "LogOutput.log"),
			fnd(KindPluginError, "Brutal Company", "Event table is empty", 7, "LogOutput.log"),
		}},
		{"valheim pack duplicates", "[Warning:   BepInEx] Skipping [Epic Loot 0.14.12] (randyknapp.mods.epicloot) at RandyKnapp-EpicLoot/EpicLoot.dll because a newer version exists (Epic Loot 0.14.13 at old/EpicLoot.dll)\n" +
			"[Warning:   BepInEx] Skipping [Jotunn 2.30.2] (com.jotunn.jotunn) at b/Jotunn.dll because a duplicate of it was already loaded from a/Jotunn.dll\n", "", []Finding{
			fnd(KindDuplicateGUID, "Epic Loot 0.14.12", "a newer copy loaded instead: Epic Loot 0.14.13 at old/EpicLoot.dll", 1, "LogOutput.log"),
			fnd(KindDuplicateGUID, "Jotunn 2.30.2", "the same copy loaded instead from a/Jotunn.dll", 2, "LogOutput.log"),
		}},
		// BepInEx's Unity log listener copies an exception Unity caught, such as one thrown in a plugin's Awake, with Unity's
		// own stack format; Player.log may be unreadable, so LogOutput.log alone has to name the plugin, and once.
		{"plugin exception through the Unity log", awake, "", []Finding{
			fnd(KindUnityException, "MortarMatrix", "InvalidOperationException: matrix probe failed in Awake on purpose", 2, "LogOutput.log"),
		}},
		{"the same exception in both logs", awake, "InvalidOperationException: matrix probe failed in Awake on purpose\n" +
			"  at MortarMatrix.MatrixProbe.Awake () [0x0007b] in <6f3c2a1e5b7d4c8a9e0f1b2c3d4e5f60>:0 \n", []Finding{
			fnd(KindUnityException, "MortarMatrix", "InvalidOperationException: matrix probe failed in Awake on purpose", 2, "LogOutput.log"),
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

func dep(f Finding, dependency string) Finding {
	f.Dependency = dependency
	return f
}

func TestUnclassified(t *testing.T) {
	log := "[Warning:  HarmonyX] AccessTools.DeclaredMethod: Could not find method for type X and name y and parameters \n" +
		"[Error  : Unity Log] NullReferenceException: Object reference not set to an instance of an object\n" +
		"[Warning:   BepInEx] Something BepInEx has not said before\n" +
		"[Error  :   BepInEx] Could not load [A 1.0.0] because it has missing dependencies: B\n"
	if n := Unclassified(log); n != 2 {
		t.Fatalf("Unclassified = %d, want 2", n)
	}
}

func fnd(kind, plugin, message string, line int, source string) Finding {
	return Finding{Kind: kind, Plugin: plugin, Message: message, Line: line, Source: source}
}
