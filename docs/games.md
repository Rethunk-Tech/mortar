# Games

Per game, the decisions and sources behind its catalog entry that the catalog cannot carry. Ids, executables, folders, save and log paths, loaders and sources live in the catalog (`internal/components/components.json`, built from `components.source.json`; `mortar games` lists it). What nobody has read from a real install is in [design.md](design.md#game-facts-to-verify).

## Cat Mail Co.

Source for the save layout: CatLib's docs (`okatodev/CatLib` `docs/Saves.md`) and the maintainer's report of a real save folder.
- Each save is one file, `GameSaves/GameSave_<date>_<time>_Cat-Mail-Co.bin`, under the game's LocalLow folder. The saves path stays that folder (where 0.3.12 put it, so a profile's own saves copy keeps its shape) and `saveFiles` reaches one folder down.
- CatLib keeps a save's mod data in `CatLibSaves/<save name>/` beside `GameSaves` (JSON per mod: `mod`, `save`, `values`; a CatLib user's report), so `saveCompanionPaths` carries that folder with its save in backups and restores. A backup made before keeps the current mod data on restore rather than deleting it.

## Cities: Skylines II

Nobody on the team owns the game. Sources: PCGamingWiki, a decompiled 1.x Steam build (`bworthy89/roadmod`), the GDK build's API dump (`ps1ke/Cities-Skylines-2-Modding-Guide`), LTT Labs' benchmark harness (`LTTLabsOSS/markbench-tests`).
- The folder below `Saves` is `PlatformManager.userSpecificPath`, a Steam id on Steam (`SaveGameMetadata.kPersistentLocation` in the decompile; markbench's `SAVE_LOCATION`). That is why the entry sets `saveUserFolders`.
- The save dialog offers `Local`, `SteamCloud` (`AssetDatabase<SteamCloud>` in `GameManager`) or on Xbox `GdkCloud`; the choice is `lastCloudTarget` in the `UserState` settings file. Autosaves always go local (`AutoSaveSystem.GetAutoSaveDatabaseTarget`).
- Swapping the local `Saves` folder per profile cannot overwrite or delete a cloud save: the app declares no Auto-Cloud paths (PCGamingWiki; Steam app info), Steam Cloud saves go through the Steam API into `userdata/<id>/949230/remote`, and Paradox account sync covers mods and playsets only (`PdxSdkPlatform.SyncModConflict`). A city saved to a cloud target stays outside the swap, shared by every profile.
- No `newProfileSeparateSaves`: a new profile would open on an empty city list, and cities run to hundreds of MB.

## Lethal Company

Sources: Gale (`Kesomannen/gale`), r2modmanPlus (`ebkr/r2modmanPlus`), the Thunderstore server source (`thunderstore-io/Thunderstore`), BepInEx 5.4.23.5. Modded from Thunderstore, Nexus, CurseForge and GitHub (the catalog's `sources`).
- **Doorstop files:** `winhttp.dll` and `doorstop_config.ini` must sit next to the game executable, so launch copies the two from the profile into the game folder and removes them after, as Gale does (`Kesomannen/gale` `src-tauri/src/profile/launch/mod.rs:183-224`, `copy_required_files`). Everything else stays in the profile because BepInEx 5 takes its root from the preloader it started with (`BepInEx.Preloader/Entrypoint.cs:61`, `BepInEx/Paths.cs:11-30`).
- **Launch arguments** follow the profile's `.doorstop_version`: `--doorstop-enable true --doorstop-target <preloader>` for 3.x, `--doorstop-enabled true --doorstop-target-assembly <preloader>` for 4.x. Mortar never writes `doorstop_config.ini`; the arguments override what the proxy reads. Under Proton the target takes the `Z:` form.
- **Proton:** the Wine override `winhttp=n,b` goes into `compatdata/1966720/pfx/user.reg`, backed up first, as Gale and r2modman do.
- **Saves** are encrypted Easy Save 3 data with no mod list, so the Saves tab lists them by slot with an unknown fit. Easy Save 3 (`Assembly-CSharp-firstpass.dll`) writes `.bac`, `.tmp` and `.tmp.bak` companions beside a slot.
- **Package layout** follows r2modman's walk (`InstallRulePluginInstaller.ts`, `buildInstallForRuleSubtype`), which mod authors build against. Two files landing in one place: the later in a folder-files-first walk wins, ignoring case; an archive with two names differing only in case is refused (r2modman keeps the later on Windows and both on Linux). MirageCore ships `FSharp.Core/FSharp.Core.dll` and Mirage loads it from beside its plugin, so folders above a loose file are flattened.
- **r2modman profile codes** are imported and exported because that is how players share: `#r2modman\n` plus base64 of an `.r2z` zip holding `export.r2x` (YAML) and `config/`, stored through `/api/experimental/legacyprofile/create/` and `get/{key}/`. The endpoint is experimental and Gale uses it too.
- **Mod identity** is the Thunderstore package. `[BepInPlugin]` GUIDs are only a per-version secondary key, since a package can hold one plugin, several or none and a GUID can change between versions.
- **Load order** is BepInEx 5.4.23's: newest copy of each GUID, GUIDs sorted case-insensitively in a `SortedDictionary`, then `Utility.TopologicalSort` (`BepInEx/Bootstrap/Chainloader.cs` 312-366, `BepInEx/Utility.cs` 135-180). A cycle makes the sort throw and no plugin loads. `TestRealBepInExRun` checks the prediction against a real run.
- **Game version:** Unity's `Application.version` is the unset `0.1`, so the bridge reports `GameNetworkManager.gameVersionNum`. MoreCompany adds 9950 to it in an `Awake` postfix, so the bridge reads it in a prefix.
- **Intro skip:** the game reads no command-line argument but Doorstop's, and the Online or LAN choice and the cold-open cinematics live in encrypted `LCGeneralSaveData`, so no launch option or registry key skips them (`Assembly-CSharp`: `InitSceneLaunchOptions`, `InitializeGame`). Only `MORTAR_SKIP_INTRO=1` from `scripts/launch-guard.sh` does, through the bridge's `IntroSkip.cs`.

## PEAK

- Campfire saves exist since the "Play It Your Way" patch, so PCGamingWiki's "no save data" is outdated.
- `Player.log` is built from PCGamingWiki's registry key `HKCU\Software\LandCrab\PEAK`.
- The `graphics` block follows r2modman issue [#1842](https://github.com/ebkr/r2modmanPlus/issues/1842) (Steam's first launch option carries `-force-vulkan`) and the BepInExPack_PEAK readme (add `-dx12` if PEAK crashes on startup).

## R.E.P.O

- Saves and `Player.log` follow PCGamingWiki's `AppData\LocalLow\semiwork\Repo\saves\`. With no `saveFiles` pattern a save is a folder holding a file of its own name.

## Risk of Rain 2

- **Dedicated server** writes `Risk of Rain 2_Data/Config/server.cfg` instead of passing flags. The file, folder and example lines are from the Steam dedicated server guide (`steamcommunity.com/sharedfiles/filedetails/?id=1938378081`, by `ghor`); the variable names are registered in the decompiled `RoR2.dll` (`WarmBuns/DecompiledPile`: `NetworkManagerSystem`, `SteamworksServerManager`), where `exec <name>` runs `Config/<name>.cfg`. The `avivace/ror2-server` image agrees.
- The loader is `bbepis-BepInExPack` because the community's mods do not depend on the generic pack and `RiskofThunder-BepInExPack` is deprecated.
- No Nexus source: Nexus does not carry the game's mods.
- No `saves` role: PCGamingWiki lists only Steam Cloud, outside any path template Mortar resolves.
- `Player.log` is built from PCGamingWiki's registry key `HKCU\Software\Hopoo Games, LLC\Risk of Rain 2`.

## The Sims 4

Each decision names its community source, because EA documents little. The catalog's `folderNames` and `requiredSettings` encode most of it.
- **Language folders:** the user folder follows the game's language and switching starts a new empty one beside the old (EA forums `11781561`; heise.de's Sims 4 mods guide). Mortar uses whichever exists and changed last.
- **Redirected Documents** (OneDrive, a moved folder, a symbolic link) is common (EA forums `11782389`), so paths resolve through the real Documents folder. Under Proton it is `compatdata/1222670/pfx/drive_c/users/steamuser/Documents/Electronic Arts/The Sims 4` (a Steam Deck user's `Mods` path, Steam discussion `4364628453802426195`).
- **Executables:** `Game/Bin` also holds `TS4_DX9_x64.exe`, the DirectX 9 build (EA forums `11819001`); the old Legacy Edition sat in `Game/Bin_LE`.
- **`Options.ini`:** `modsdisabled` must be `0` and `scriptmodsenabled` `1` (EA forums `8908087`). Patches turn the in-game switches off (help.ea.com, "Mods and The Sims 4 game updates") and the game rewrites the file, so script mods can reset to off after a restart (EA forums `13525568`). Hence Mortar keeps the setting in the profile's copy and checks it before Play.
- **Rollback copies:** `Slot_########.save` sits beside `.save.ver0` (newest) to `.ver4` (oldest), which are not saves of their own; removing the `.verN` suffix restores one (EA's backup guide; simscommunity.info).
- **Depth limits:** `Resource.cfg` allows packages five folders below `Mods` (bank42n.com's guide); the game creates it when missing and the stock text is a `Priority 500` block of `PackedFile` lines that CC managers extend. `.ts4script` works only one folder below `Mods` (community guidance; EA gives no figure).
- **Caches:** `localthumbcache.package` is safe to delete and rebuilt on launch, advised after mod changes; `avatarcache.package` is safe too, for Gallery errors (thesimstree.com, carls-sims-4-guide.com).

## Valheim

Sources: r2modman's ecosystem schema, MO2's `games/game_valheim.py`, Vortex's `Nexus-Mods/game-valheim`, the `denikson-BepInExPack_Valheim` zip.
- **Native Linux launch** is always direct, because Steam's own start cannot carry the environment: Mortar does what the pack's Linux start script does (`LD_PRELOAD` the profile's `doorstop_libs/libdoorstop_x64.so`, `DOORSTOP_ENABLED=1`, `DOORSTOP_TARGET_ASSEMBLY`) plus `SteamAppId` so Steamworks does not restart the game through Steam. The shared `BepInEx-BepInExPack` ships no Linux library, so this works only with a community pack that has one.
- **Loader pack:** the community's own `denikson-BepInExPack_Valheim`; the generic pack is not listed in the community. Its files sit under `BepInExPack_Valheim/` in the zip, which `InstallPack` finds by the folder holding the preloader.
- Mods name the pack as a dependency, which Mortar skips because the loader install provides it.
- **Steam Cloud saves** live in `userdata/<id>/892970/remote/`, outside the `saves` path.
- **Logs:** the pack's `Chainloader startup complete` line ends with counts, which `Ready` still matches. Duplicate-GUID skips add the GUID and both paths, and an identical second copy reads `because a duplicate of it was already loaded from <path>` (`AzumattDev/BepInEx` `Chainloader.cs`); the analyzer reads both wordings.
- **Dedicated server** (`valheim-server`) refuses to start without a password of five or more characters, which is the game's rule.

## Vintage Story

- **AUR** installs to `/opt/vintagestory` but is not in the catalog, because a path template cannot start with a token; the player chooses the folder.
- **Linux start** goes through `run.sh` (`linuxStart`), which points fontconfig at the game's `fonts.conf` and sets the working folder. A .NET runtime installed by Microsoft's script lives in `~/.dotnet`, which the desktop leaves out of `DOTNET_ROOT`, so Mortar sets it when unset and that runtime exists.
- **Saves:** a restore deletes `-shm` and `-wal` companions the restored copy lacks, since SQLite replays a leftover `-wal` over it. Each world's `gamedata` row is a protobuf `SaveGame` ([vs-proto](https://github.com/bluelightning32/vs-proto) schema); Saves reads it read-only and immutable for name, play style, play time and version.
- **Logs:** names and level words are from `VintagestoryLib.dll`; the mod loader's failure lines (`internal/loader/vslog`) come from logs users pasted into GitHub issues. The game keeps one log for every profile, so Problems reads it only for the profile that ran last.
- **Mods in a world:** a world's `ModData` holds `BlockIDs` and `ItemIDs` whose codes carry the mod id as domain (`primitivesurvival:fishbasket`; the game's own are bare or `creative`/`survival`). A mod of code alone leaves no trace, so the world stays marked as not recording its full mod list.
- **`--addModPath` and `--dataPath`** are unused: the folder deployer fills and empties the Mods folder per launch.
- **Mod DB** (`internal/source/vsmoddb`): `/mods` takes a `limit` and no offset; a mod's page is `/show/mod/<assetid>` or `/<urlalias>` and the API does not answer to the assetid, so Mortar's id is `<modid>-<assetid>`. Releases carry no hash and no dependency list, so after an install Mortar reads each `modinfo.json` `dependencies` and queues the required mods missing from the profile.
- **Install links:** the site's 1-click button opens `vintagestorymodinstall://<modid>@<version>`, which the game's installer registers for `Vintagestory -i`. Mortar claims the scheme only when the player turns it on.
- **Settings:** the folder loader reads every per-game setting as for The Sims 4, except `skipIntro` and `graphicsApi` (no catalog entry), `gameSettingsMode` (no `requiredSettings`), `defaultLaunchMethod` (no store relay: every copy starts directly), and `enableRequirements` and `missingRequirements` (a Mod DB install queues the mods it requires).
