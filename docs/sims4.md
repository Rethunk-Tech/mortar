# The Sims 4: catalog facts

Each fact carries its source and one of three states: confirmed (a primary or near-primary source states it), unconfirmed (only community posts, or no source reached), contradicted (a source says otherwise). EA Help pages answered slowly or not at all, several EA forum threads answered 403 to a direct fetch and were read through search summaries only, and PCGamingWiki answered 403, so nothing below rests on a page read in full unless it says so.

| Fact | State | Source |
| --- | --- | --- |
| Steam app id `1222670`, a Windows-only game that needs the EA app | confirmed | <https://store.steampowered.com/api/appdetails?appids=1222670> |
| Steam install folder `steamapps/common/The Sims 4` | confirmed | <https://steamcommunity.com/app/1222670/discussions/0/3003299213533445281> (the executable sits at steamapps, common, The Sims 4, Game, Bin) |
| EA App install folder `The Sims 4` under `EA Games` (`C:\Program Files\EA Games\The Sims 4`) | confirmed | <https://forums.ea.com/discussions/the-sims-4-technical-issues-pc-en/i-cant-find-my-origin-games-file/11811742> |
| Executable `Game/Bin/TS4_x64.exe` | confirmed for the path, unconfirmed for letter case | <https://answers.ea.com/t5/General-Discussion-Feedback/where-to-find-sims-4-game-bin-on-ea-desktop/m-p/10441531>; case is as the catalog writes it, and the match is case-sensitive on Linux |
| A DX9 or legacy executable beside it | unconfirmed | no source found naming one |
| Documents path `Documents/Electronic Arts/The Sims 4`, holding `Mods`, `saves` and `Options.ini` | confirmed for `Mods` | <https://blog.usro.net/2025/04/where-is-the-mods-folder-in-the-sims-4/>; `saves` and `Options.ini` as siblings are unconfirmed beyond community guides |
| Saves are `Slot_########.save` files in `saves`, beside the game's own `.save.ver0` to `.ver4` rollback copies, which are not saves of their own | unconfirmed (the catalog's `saveFiles` is `Slot_*.save`, written from memory of an install) | none |
| Under Proton the same folder is `compatdata/1222670/pfx/drive_c/users/steamuser/Documents/Electronic Arts/The Sims 4` | confirmed | <https://steamcommunity.com/app/1222670/discussions/0/4364628453802426195> (a Steam Deck user gives the `Mods` path there) |
| Documents may be redirected (OneDrive, a moved Documents folder, a symbolic link) | confirmed | <https://forums.ea.com/discussions/the-sims-4-technical-issues-pc-en/mods-does-not-show-up-in-game/11782389> |
| The folder name is localized on non-English installs ("Die Sims 4", "Les Sims 4") | unconfirmed | a search for localized names found only the English path; Mortar's catalog carries the English name only |
| `Options.ini` file name, with the keys `modsdisabled` (needs `0`) and `scriptmodsenabled` (needs `1`) | confirmed for the keys, unconfirmed for the section name | <https://forums.ea.com/discussions/the-sims-4-technical-issues-es/script-mods-no-se-mantiene-habilitado/8908087> (a Spanish thread names both keys and values); the `[options]` section header is not shown by any source reached |
| The in-game switches are Game Options, Other, "Enable Custom Content and Mods" and "Script Mods Allowed"; patches turn the first off | confirmed | <https://help.ea.com/en-us/help/the-sims/the-sims-4/mods-and-the-sims-4-game-updates> |
| The game rewrites `Options.ini` and can reset script mods to off after a restart | confirmed as reported | <https://forums.ea.com/discussions/the-sims-4-technical-issues-pc-en/script-mods-setting-keeps-resetting-to-off-after-restart-sims-4-version-1-125/13525568> |
| `Resource.cfg` is created in `Mods` by the game when missing | confirmed | <https://modthesims.info/d/694839/mods-folder-broom.html> and community guides |
| `Resource.cfg` allows packages up to five folders below `Mods` | confirmed | <https://bank42n.com/docs/2-general-guides/resourcecfg> |
| The stock `Resource.cfg` text | unconfirmed | no source quotes it; guides show a priority block of `PackedFile *.package` patterns, `*/*.package` and so on to five levels |
| `.ts4script` works only one folder below `Mods` | confirmed as community guidance | <https://www.patreon.com/posts/71648623>; EA gives no figure |
| `localthumbcache.package` is safe to delete and is rebuilt on launch; deleting it after adding, removing or updating mods is advised | confirmed as community guidance | <https://www.thesimstree.com/en/blog/the-sims-tips/clear-sims-4-cache-fix-lags,-errors-and-slow-loading-(windows-and-macos).html>, <https://www.carls-sims-4-guide.com/help/cache.php> |
| `avatarcache.package` is safe to delete, with a narrower purpose (Gallery errors) | confirmed as community guidance | same pages |

## What the catalog holds

`internal/components/components.json` `sims4` matches every confirmed row: app id, EA folder `The Sims 4`, `markerDir` `Game/Bin` with marker `TS4_x64.exe`, the three paths under `{documents}/Electronic Arts/The Sims 4`, `maxDepth` 5 for `package` and 1 for `ts4script`, and `modsdisabled = 0` and `scriptmodsenabled = 1`. The depth counts folders below the target root. The catalog stays `enabled: false`.

## Unverified

- The `Options.ini` section header, and whether the game matches key names case-insensitively.
- A DX9 or legacy executable.
- Localized Documents folder names.
- The stock `Resource.cfg` text.
- An EA-published statement of the `.ts4script` depth limit.
- The CurseForge class ids and the game's behaviour were measured live and are not listed here.
