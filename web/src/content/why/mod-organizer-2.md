# Why Mortar? (vs Mod Organizer 2)
<!-- tile: The usual pick for heavily modded Bethesda games. -->

Mod Organizer 2 is the usual choice for heavily modded Bethesda games; Mortar reads its profiles. Use Mortar for games MO2 supports only lightly, native Linux, or one search across mod sites.

| | Mortar | Mod Organizer 2 |
| --- | --- | --- |
| Platforms | !win Windows, Linux, Steam Deck | Windows ([MO2_RELEASE](https://github.com/ModOrganizer2/modorganizer/releases/latest)); Linux through a [third-party Wine installer](https://github.com/rockerbacon/modorganizer2-linux-installer) |
| Games | GAME_COUNT, each with its loader installed | MO2FULL_COUNT fully supported Bethesda games, MO2BASIC_COUNT with basic support ([Game Support](https://github.com/ModOrganizer2/modorganizer/wiki/Game-Support)) |
| Mod sites | !win MORTAR_SITES | :nexus: and local archives ([source](https://github.com/ModOrganizer2/modorganizer/blob/master/src/nexusinterface.h)) |
| Loader install | !win SMAPI, BepInEx and each game's loader | Set up by hand ([wiki](https://github.com/ModOrganizer2/modorganizer/wiki/Running-Oblivion-OBSE-with-MO2)) |
| Sharing | !win A link or a `.mortar` file | None documented |
| Latest stable release | !win MORTAR_VERSION ([changelog](/changelog/)) | MO2_RELEASE, MO2_RELEASED |
| Conflict tools | Content Patcher conflicts explained in Problems | !win Priority, winning and losing files, hidden files, LOOT ([comparison](https://github.com/ModOrganizer2/modorganizer/wiki/Mod-Managers-Comparison)) |
| Licence | AGPL-3.0 | GPL-3.0 |

## Where Mod Organizer 2 is stronger

- Its virtual file system keeps the game folder untouched and switches profiles instantly ([comparison](https://github.com/ModOrganizer2/modorganizer/wiki/Mod-Managers-Comparison)).
- Conflict and load order tools for Bethesda games: priority, winning and losing files, hidden files and LOOT.

## Moving from Mod Organizer 2

Profiles → **Import…**: Mortar lists MO2 profiles for the game, with a preview. MO2's folders are read, never changed.
