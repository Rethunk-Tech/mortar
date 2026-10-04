// Stand-ins for the game's types, so the fixture mod compiles to the same member references a real mod has.
namespace Netcode
{
    public class NetFieldBase<T, TSelf> where TSelf : NetFieldBase<T, TSelf>
    {
        public T Value { get; set; }
    }

    public sealed class NetInt : NetFieldBase<int, NetInt>
    {
    }
}

namespace StardewValley
{
    using Netcode;

    public class Farmer
    {
        public readonly NetInt health = new NetInt();
        public readonly NetInt maxHealth = new NetInt();
        public int stamina;

        public int CurrentToolIndex { get; set; }

        public NetInt NetMoney { get; } = new NetInt();

        public class Pocket
        {
            public int Count { get; set; }
        }
    }

    public static class Game1
    {
        public static Farmer player = new Farmer();
        public static float flashAlpha;

        public static Menus.ClickableComponent CurrentComponent { get; } = new Menus.ClickableComponent();
    }
}

namespace StardewValley.Buffs
{
    using Netcode;

    public class BuffEffects
    {
        public readonly NetInt Defense = new NetInt();

        public int Speed { get; set; }
    }
}

namespace StardewValley.Menus
{
    public class ClickableComponent
    {
        public int myID;
        public int leftNeighborID;
        public int rightNeighborID;
        public int upNeighborID;
    }

    public class ClickableTextureComponent : ClickableComponent
    {
    }
}
