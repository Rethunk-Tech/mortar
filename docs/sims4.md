# The Sims 4: catalog facts

Each fact carries its source and a state: confirmed (a primary or near-primary source), unconfirmed (community posts only, or none reached), contradicted.

| Fact | State | Source |
| --- | --- | --- |
| Steam app id `1222670`, a Windows-only game that needs the EA app | confirmed | <https://store.steampowered.com/api/appdetails?appids=1222670> |
| Steam install folder `steamapps/common/The Sims 4` | confirmed | <https://steamcommunity.com/app/1222670/discussions/0/3003299213533445281> (the executable sits at steamapps, common, The Sims 4, Game, Bin) |
| EA App install folder `The Sims 4` under `EA Games` (`C:\Program Files\EA Games\The Sims 4`) | confirmed | <https://forums.ea.com/discussions/the-sims-4-technical-issues-pc-en/i-cant-find-my-origin-games-file/11811742> |
| Executable `Game/Bin/TS4_x64.exe` | path confirmed, letter case not | <https://answers.ea.com/t5/General-Discussion-Feedback/where-to-find-sims-4-game-bin-on-ea-desktop/m-p/10441531>; the match is case-sensitive on Linux |
| `Game/Bin` also holds `TS4_DX9_x64.exe`, the DirectX 9 build, beside the DirectX 11 `TS4_x64.exe`; the old Legacy Edition sat in `Game/Bin_LE` | confirmed as reported | <https://forums.ea.com/discussions/-/-/11819001>, <https://steamcommunity.com/app/1222670/discussions/0/4849903159943871190> |
| Documents path `Documents/Electronic Arts/The Sims 4`, holding `Mods`, `saves` and `Options.ini` | confirmed for `Mods` | <https://blog.usro.net/2025/04/where-is-the-mods-folder-in-the-sims-4/>; `saves` and `Options.ini` as siblings are unconfirmed beyond community guides |
| Saves are `Slot_########.save` files in `saves`, beside the game's own `.save.ver0` (newest) to `.ver4` (oldest) rollback copies, which are not saves of their own; removing the `.verN` suffix restores one | confirmed (community, EA forums); the catalog's `saveFiles` is `Slot_*.save` and `saveCopies` is `{name}.ver?` | <https://forums.ea.com/blog/the-sims-game-info-hub-en/backing-up-saves-in-the-sims-4---keep-your-sims-safe/12913482>, <https://simscommunity.info/2015/02/13/sims-4-tutorial-restore-previous-saved-game/> |
| Under Proton the same folder is `compatdata/1222670/pfx/drive_c/users/steamuser/Documents/Electronic Arts/The Sims 4` | confirmed | <https://steamcommunity.com/app/1222670/discussions/0/4364628453802426195> (a Steam Deck user gives the `Mods` path there) |
| Documents may be redirected (OneDrive, a moved Documents folder, a symbolic link) | confirmed | <https://forums.ea.com/discussions/the-sims-4-technical-issues-pc-en/mods-does-not-show-up-in-game/11782389> |
| The user folder follows the game's language: `Die Sims 4` (German), `Les Sims 4` (French), `Los Sims 4` (Spanish), `De Sims 4` (Dutch), `The Sims 4` for every other; switching language starts a new empty folder beside the old one | confirmed (EA forums, a German guide) | <https://forums.ea.com/discussions/the-sims-4-technical-issues-pc-en/re-change-game-language/11781561>, <https://www.heise.de/tipps-tricks/Sims-4-Mods-installieren-so-klappt-s-10200610.html> |
| `Options.ini` file name, with the keys `modsdisabled` (needs `0`) and `scriptmodsenabled` (needs `1`) | confirmed for the keys, unconfirmed for the section name | <https://forums.ea.com/discussions/the-sims-4-technical-issues-es/script-mods-no-se-mantiene-habilitado/8908087> (a Spanish thread names both keys and values); the `[options]` section header is not shown by any source reached |
| The in-game switches are Game Options, Other, "Enable Custom Content and Mods" and "Script Mods Allowed"; patches turn the first off | confirmed | <https://help.ea.com/en-us/help/the-sims/the-sims-4/mods-and-the-sims-4-game-updates> |
| The game rewrites `Options.ini` and can reset script mods to off after a restart | confirmed as reported | <https://forums.ea.com/discussions/the-sims-4-technical-issues-pc-en/script-mods-setting-keeps-resetting-to-off-after-restart-sims-4-version-1-125/13525568> |
| `Resource.cfg` is created in `Mods` by the game when missing | confirmed | <https://modthesims.info/d/694839/mods-folder-broom.html> and community guides |
| `Resource.cfg` allows packages up to five folders below `Mods` | confirmed | <https://bank42n.com/docs/2-general-guides/resourcecfg> |
| The stock `Resource.cfg` text is a `Priority 500` block of `PackedFile` lines, one per depth from `*.package` to `*/*/*/*/*.package`; CC managers add blocks of their own | confirmed as community guidance | <https://bank42n.com/docs/2-general-guides/resourcecfg>, The Sims Resource's help page on the file |
| `.ts4script` works only one folder below `Mods` | confirmed as community guidance | <https://www.patreon.com/posts/71648623>; EA gives no figure |
| `localthumbcache.package` is safe to delete and is rebuilt on launch; deleting it after adding, removing or updating mods is advised | confirmed as community guidance | <https://www.thesimstree.com/en/blog/the-sims-tips/clear-sims-4-cache-fix-lags,-errors-and-slow-loading-(windows-and-macos).html>, <https://www.carls-sims-4-guide.com/help/cache.php> |
| `avatarcache.package` is safe to delete, with a narrower purpose (Gallery errors) | confirmed as community guidance | same pages |

## Catalog and open checks

The catalog entry (`internal/components/components.json` `sims4`) matches every confirmed row; `folderNames` holds the four localized folder names and Mortar uses whichever exists and changed last. Depths count folders below the target root.

Unverified, and tracked in [design.md](design.md#the-sims-4-before-a-release-carries-it): the `Options.ini` section header and key-name case (Mortar's check finds the keys in any section, ignoring case), and an EA statement of the `.ts4script` depth limit.
