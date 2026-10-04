using StardewValley;

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

        public int Read(Farmer farmer) => farmer.CurrentToolIndex + farmer.maxHealth.Value + this.own;
    }
}
