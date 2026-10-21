package dotnetstack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBlamesReadsEachRuntimesFrameFormat(t *testing.T) {
	cases := []struct {
		name, log string
		notMod    map[string]bool
		ns        string
		frame     string
	}{
		{"BepInEx Unity Log", `[Info   :   BepInEx] Loading [Host Fixes 1.0.28]
[Error  : Unity Log] NullReferenceException: Object reference not set to an instance of an object
Stack trace:
GameNetcodeStuff.PlayerControllerB.Update () (at <9e2a4c3b>:0)
HostFixes.Plugin+Patches.Fix (GameNetcodeStuff.PlayerControllerB __instance) (at <474744d6>:0)
UnityEngine.SendMouseEvents:DoSendMouseEvents(Int32)
`, Unity, "HostFixes", "HostFixes.Plugin+Patches.Fix (GameNetcodeStuff.PlayerControllerB __instance)"},
		{"Mono Player.log", `NullReferenceException: Object reference not set to an instance of an object
  at UnityEngine.GameObject.GetComponent[T] () [0x00000] in <abc>:0
  at CoolMod.Behaviours.Spawner.Awake () [0x0001f] in <def>:0
  at BepInEx.Bootstrap.Chainloader.Start () [0x00000] in <ghi>:0
`, Unity, "CoolMod.Behaviours", "at CoolMod.Behaviours.Spawner.Awake ()"},
		{"MelonLoader", `[12:01:02.345] [ERROR] System.InvalidOperationException: Sequence contains no elements
   at System.Linq.Enumerable.First[TSource](IEnumerable` + "`1" + ` source)
   at BetterHands.Core.OnUpdate() in C:\src\Core.cs:line 41
   at MelonLoader.MelonBase.OnUpdate()
`, Unity, "BetterHands", "at BetterHands.Core.OnUpdate()"},
		{"Vintage Story crash report", `4.10.2026 19:20:01 [Fatal] Caught unhandled exception in thread 'Main'
System.NullReferenceException: Object reference not set to an instance of an object.
   at Vintagestory.API.Common.Block.GetDrops(IWorldAccessor world)
   at primitivesurvival.BlockFishBasket.OnBlockInteractStart(IWorldAccessor world) in /src/Block.cs:line 88
   at Vintagestory.Client.NoObf.ClientMain.Run()
`, VintageStory, "primitivesurvival", "at primitivesurvival.BlockFishBasket.OnBlockInteractStart(IWorldAccessor world)"},
	}
	for _, c := range cases {
		got := Blames(c.log, c.notMod)
		if len(got) != 1 || got[0].Namespace != c.ns || got[0].Frame != c.frame && got[0].Frame != "at "+c.frame {
			t.Errorf("%s: got %+v, want namespace %q frame %q", c.name, got, c.ns, c.frame)
		}
	}
}

func TestBlamesIgnoresStacksWithoutAModFrame(t *testing.T) {
	log := "NullReferenceException: x\nStack trace:\nUnityEngine.GameObject.Find (System.String name) (at <a>:0)\nTMPro.TMP_Text.Update () (at <b>:0)\n"
	if got := Blames(log, Unity); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}

// Real stacks from a Lethal Company run, one per block: Unity's format through MonoMod hooks (LethalCasino), Mono's
// Exception.ToString with source lines and symbol-less frames (FacilityMeltdown), a patch without symbols inside its
// Harmony wrapper (TooManyEmotes), and HarmonyX's own report of a patch it could not run. Each stack ends at the
// blank line after it.
func TestFramesReadsRealStacks(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "lc-stacks.log"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(data), "\n")
	var stacks [][]Frame
	for i := range lines {
		if i == 0 || strings.TrimSpace(lines[i-1]) == "" && strings.TrimSpace(lines[i]) != "" {
			stacks = append(stacks, Frames(lines[i+1:]))
		}
	}
	casino, meltdown, emotes, harmonyx := 0, 1, 2, 3
	if len(stacks) != 4 {
		t.Fatalf("%d stacks", len(stacks))
	}
	for _, c := range []struct {
		stack, frames int
		at            int
		want          Frame
	}{
		{casino, 23, 0, Frame{Method: "LethalCasino.Patches.MenuManagerPatch::StartPatch", Source: "MenuManagerPatch.cs:17"}},
		{casino, 23, 1, Frame{Method: "MenuManager::Start", DMD: true}},
		{casino, 23, 2, Frame{Method: "MonoMod.Utils.DynamicMethodDefinition::Trampoline<MenuManager::Start>?659680686"}},
		{casino, 23, 21, Frame{Method: "Mirage.Hook.Config+syncConfig@23-1::Invoke", Source: "Config.fs:24"}},
		{meltdown, 8, 0, Frame{Method: "FacilityMeltdown.MeltdownSequence.Behaviours.MeltdownHandler+<>c__DisplayClass20_0::<HandleEnemies>b__0", Source: "MeltdownHandler.cs:180"}},
		{meltdown, 8, 1, Frame{Method: "System.Linq.Enumerable+WhereListIterator`1[TSource]::ToList", Module: "ea489521390f48908a7ce45cb17a6848"}},
		{meltdown, 8, 5, Frame{Method: "FacilityMeltdown.MeltdownSequence.Behaviours.MeltdownHandler::__rpc_handler_3631067672", Module: "f723873df70c4b288182bf3c0363ab12"}},
		{emotes, 8, 0, Frame{Method: "TooManyEmotes.SaveManager::SaveLocalPlayerValues", Module: "ba9a980d78984b3fb839a44a6ac6a81c"}},
		{emotes, 8, 1, Frame{Method: "GameNetworkManager::SaveLocalPlayerValues", DMD: true}},
		{harmonyx, 1, 0, Frame{Method: "RoundManager::GenerateNewFloor", DMD: true}},
	} {
		got := stacks[c.stack]
		if len(got) != c.frames || got[c.at] != c.want {
			t.Errorf("stack %d: %d frames, [%d] = %+v, want %d and %+v", c.stack, len(got), c.at, got[min(c.at, len(got)-1)], c.frames, c.want)
		}
	}
	blames := Blames(string(data), Unity)
	if len(blames) < 3 || blames[2].Frame != "at TooManyEmotes.SaveManager.SaveLocalPlayerValues ()" {
		t.Errorf("a frame's symbol-less location is cut from its blame: %+v", blames)
	}
}
