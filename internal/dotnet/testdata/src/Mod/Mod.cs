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
    public class Plugin
    {
    }
}
