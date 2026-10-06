using System.Collections.Generic;
using StardewValley;
using StardewValley.Buffs;
using StardewValley.Menus;

namespace Fixture
{
    public class Mod
    {
        private int own;

        public int Own { get; set; }

        public void SwitchTool(Farmer farmer, int kind)
        {
            switch (kind)
            {
                case 0: farmer.CurrentToolIndex = 1; break;
                case 1: farmer.CurrentToolIndex = 2; break;
                case 2: farmer.CurrentToolIndex = 4; break;
                default: this.own = kind; break;
            }
            this.Own = kind;
        }

        public void Heal(Farmer farmer)
        {
            farmer.health.Value = farmer.maxHealth.Value;
            farmer.NetMoney.Value += 10;
            farmer.stamina = 270;
            Game1.flashAlpha = 1f;
        }

        public void Pockets(Farmer.Pocket pocket) => pocket.Count = 3;

        public BuffEffects Effects() => new BuffEffects { Speed = 2, Defense = { Value = 3 } };

        public ClickableComponent Component(List<ClickableComponent> into)
        {
            var built = new ClickableTextureComponent();
            built.myID = 1;
            into.Add(built);
            built.leftNeighborID = 2;
            return built;
        }

        public void Neighbours(bool current)
        {
            Game1.CurrentComponent.upNeighborID = 3;
            var either = new ClickableComponent();
            if (current)
            {
                either = Game1.CurrentComponent;
            }
            either.rightNeighborID = 4;
        }

        public int Read(Farmer farmer) => farmer.CurrentToolIndex + farmer.maxHealth.Value + this.own;
    }
}

namespace Fixture.Plugins
{
    [BepInEx.BepInPlugin("com.fixture.plugin", "Fixture Plugin", "1.2.3")]
    [BepInEx.BepInDependency("com.fixture.hard")]
    [BepInEx.BepInDependency("com.fixture.min", "2.1.0")]
    [BepInEx.BepInDependency("com.fixture.soft", BepInEx.DependencyFlags.SoftDependency)]
    [BepInEx.BepInIncompatibility("com.fixture.clash")]
    public class Plugin
    {
    }
}

namespace Fixture.Patches
{
    using System.Collections.Generic;
    using HarmonyLib;
    using StardewValley;

    [HarmonyPatch(typeof(Farmer))]
    public static class FarmerPatches
    {
        [HarmonyPatch("Update")]
        [HarmonyPrefix]
        public static bool SkipUpdate() => false;

        [HarmonyPatch(nameof(Farmer.CurrentToolIndex), MethodType.Getter)]
        public static void Postfix()
        {
        }
    }

    [HarmonyPatch(typeof(Game1), "Draw", typeof(int))]
    public static class DrawPatch
    {
        public static IEnumerable<object> Transpiler(IEnumerable<object> code) => code;

        public static void Prefix()
        {
        }
    }

    [HarmonyPatch("StardewValley.Menus.ClickableComponent", "snap")]
    public static class ByName
    {
        [HarmonyFinalizer]
        public static void Done()
        {
        }
    }

    [HarmonyPatch(typeof(Farmer), MethodType.Constructor)]
    public static class Built
    {
        public static void Postfix()
        {
        }
    }

    public static class NotAPatch
    {
        public static void Prefix()
        {
        }
    }
}
