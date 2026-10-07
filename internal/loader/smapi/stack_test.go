package smapi

import "testing"

const crashLog = `[14:20:01 INFO  SMAPI] SMAPI 4.5.2 with Stardew Valley 1.6.15 on Linux
[14:20:03 INFO  SMAPI] Loaded 2 mods:
[14:20:03 INFO  SMAPI]    Content Patcher 2.7.0 by Pathoschild
[14:20:03 INFO  SMAPI]    Farm Type Manager 1.20.0 by Esca
[14:31:44 ERROR SMAPI] An error occurred in the base update loop: System.InvalidOperationException: Collection was modified ---> System.NullReferenceException: Object reference not set to an instance of an object.
   at FarmTypeManager.ModEntry.SpawnTimer(Object sender, UpdateTickedEventArgs e) in C:\build\FarmTypeManager\ModEntry.cs:line 88
   at StardewModdingAPI.Framework.Events.ManagedEvent` + "`1" + `.Raise(TEventArgs args)
   --- End of inner exception stack trace ---
   at StardewModdingAPI.Framework.SCore.OnPlayerInstanceUpdating(SGame instance, GameTime gameTime, Action runUpdate)
   at StardewValley.Game1.Update(GameTime gameTime)
[14:31:44 ERROR SMAPI] The game crashed.
`

func TestStackBlameNamesTheInnermostModFrame(t *testing.T) {
	got := (Loader{}).StackBlame(crashLog)
	if len(got) != 1 {
		t.Fatalf("blames = %+v", got)
	}
	b := got[0]
	if b.Namespace != "FarmTypeManager" || b.Frame != "at FarmTypeManager.ModEntry.SpawnTimer(Object sender, UpdateTickedEventArgs e)" ||
		b.Exception != "System.NullReferenceException: Object reference not set to an instance of an object." {
		t.Fatalf("blame = %+v", b)
	}
}

func TestStackBlameSkipsFrameworkFramesAndOuterOnlyStacks(t *testing.T) {
	log := `[10:00:00 ERROR SMAPI] Game failed.
System.ArgumentException: bad
   at System.Collections.Generic.Dictionary` + "`2" + `.Insert(TKey key)
   at StardewValley.Game1.Update(GameTime gameTime)
   at StardewModdingAPI.Framework.SGame.Update(GameTime gameTime)
`
	if got := (Loader{}).StackBlame(log); len(got) != 0 {
		t.Fatalf("blames = %+v", got)
	}
}

func TestStackBlameFollowsAHarmonyPatchToItsOwner(t *testing.T) {
	log := `[10:00:00 ERROR SMAPI] Patched method failed. Harmony ID: Esca.FarmTypeManager
System.NullReferenceException: Object reference not set to an instance of an object.
   at DMD<StardewValley.Object::draw>(Object this, SpriteBatch b)
   at StardewValley.Object.draw_Patch1(SpriteBatch b)
   at StardewValley.Game1.Draw(GameTime gameTime)
`
	got := (Loader{}).StackBlame(log)
	if len(got) != 1 || got[0].PatchOwner != "Esca.FarmTypeManager" || got[0].Namespace != "" {
		t.Fatalf("blames = %+v", got)
	}
}
