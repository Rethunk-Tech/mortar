using StardewValley;
using StardewValley.Buffs;
using StardewValley.Menus;

// One source, built once per -p:Fixture=NAME. Small and Large write the same five game members and Large six more,
// so the smaller footprint sits inside the larger; Filler pads a profile past the Problems check's small-profile
// cutoff without sharing a member with either.
namespace Fixture
{
#if Small || Large
    public class Writer
    {
        public void Write(Farmer farmer, Farmer.Pocket pocket, BuffEffects effects)
        {
            farmer.CurrentToolIndex = 1;
            farmer.stamina = 2;
            Game1.flashAlpha = 3f;
            pocket.Count = 4;
            effects.Speed = 5;
#if Large
            farmer.health.Value = 6;
            farmer.NetMoney.Value = 7;
            effects.Defense.Value = 8;
            Game1.CurrentComponent.leftNeighborID = 9;
            Game1.CurrentComponent.rightNeighborID = 10;
            Game1.CurrentComponent.upNeighborID = 11;
#endif
        }
    }
#endif
#if Filler
    public class Writer
    {
        public void Write(ClickableComponent component) => component.myID = 1;
    }
#endif
#if Alpha
    [BepInEx.BepInPlugin("com.e2e.alpha", "E2E Alpha", "1.0.0")]
    [BepInEx.BepInIncompatibility("com.e2e.beta")]
    public class Plugin
    {
    }
#endif
#if Beta
    [BepInEx.BepInPlugin("com.e2e.beta", "E2E Beta", "1.0.0")]
    public class Plugin
    {
    }
#endif
}
