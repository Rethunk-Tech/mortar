# Games

Per game, the decisions and sources behind its catalog entry that the catalog cannot carry. Everything the catalog carries (`mortar games` lists it) is not repeated here. What nobody has read from a real install is in [design.md](design.md#lethal-company-facts-to-verify).

## Cat Mail Co.

Source for the save layout: CatLib's docs (`okatodev/CatLib` `docs/Saves.md`) and the maintainer's report of a real save folder.
- Each save is one file, `GameSaves/GameSave_<date>_<time>_Cat-Mail-Co.bin`, under the game's LocalLow folder. The saves path is that folder and `saveFiles` reaches one folder down.
- CatLib keeps a save's mod data in `CatLibSaves/<save name>/` beside `GameSaves` (JSON per mod: `mod`, `save`, `values`; a CatLib user's report), so `saveCompanionPaths` carries that folder with its save in backups and restores.

## Cities: Skylines II

Nobody on the team owns the game. Sources: PCGamingWiki, a decompiled 1.x Steam build (`bworthy89/roadmod`), the GDK build's API dump (`ps1ke/Cities-Skylines-2-Modding-Guide`), LTT Labs' benchmark harness (`LTTLabsOSS/markbench-tests`).
- The folder below `Saves` is `PlatformManager.userSpecificPath`, a Steam id on Steam (`SaveGameMetadata.kPersistentLocation` in the decompile; markbench's `SAVE_LOCATION`). That is why the entry sets `saveUserFolders`.
- The save dialog offers `Local`, `SteamCloud` (`AssetDatabase<SteamCloud>` in `GameManager`) or on Xbox `GdkCloud`; the choice is `lastCloudTarget` in the `UserState` settings file. Autosaves always go local (`AutoSaveSystem.GetAutoSaveDatabaseTarget`).
- Swapping the local `Saves` folder per profile cannot overwrite or delete a cloud save: the app declares no Auto-Cloud paths (PCGamingWiki; Steam app info), Steam Cloud saves go through the Steam API into `userdata/<id>/949230/remote`, and Paradox account sync covers mods and playsets only (`PdxSdkPlatform.SyncModConflict`). A city saved to a cloud target stays outside the swap, shared by every profile.
- No `newProfileSeparateSaves`: a new profile would open on an empty city list, and cities run to hundreds of MB.
- **Paradox Mods:** the `pdx` target (path role `pdxMods`) names the `paradox` source, so the `source-target` installer runs before every other driver and lays the archive out whole below `paradox-<mod id>`, whatever the BepInEx rules would make of its DLLs. Placed files are recorded per entry with their hashes, removed with it, and never replace a file nobody placed (as Tray files).

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

## Stardew Valley

Sources: SMAPI's repository (technical docs, manifest and scanner models, unix launcher) and the Stardew Valley wiki's Modding pages.

### Finding the game

- Steam app `413150`, through `libraryfolders.vdf`. Per-account files such as `localconfig.vdf` are read for the account marked `MostRecent` in `config/loginusers.vdf`. Native Steam is preferred over Flatpak Steam.
- **Flatpak Steam:** its sandbox cannot read Mortar's data folder until `flatpak override --user --filesystem=<data dir>:ro com.valvesoftware.Steam`.
- **Epic (Windows):** Play opens `com.epicgames.launcher://apps/<AppName>?action=launch&silent=true`, as r2modman and Gale do, so the launcher signs the game in.
- **Xbox (Windows):** the install folder of the catalog's package name is read from the per-user AppModel registry (`internal/gamestore/appx_windows.go`), after the name is checked as a plain package name; no PowerShell runs.
- **Heroic and Lutris (Linux):** the game runs in the prefix its launcher names. Play hands off to the launcher when the plan needs only DLL overrides, which Mortar writes into the prefix's `user.reg` first. GOG, Minigalaxy, Heroic and Lutris start SMAPI directly with `--mods-path`.
- **Reset game install** deletes the validated install folder (SMAPI and leftover mods included); saves and profiles stay. It is refused while the game runs and when the path is the filesystem root or the user's home.
- **Steam Deck:** install in Desktop Mode; in Game Mode add Mortar as a non-Steam game and grant Flatpak Steam read access to Mortar's data folder.

### SMAPI

- **Pin and sweep:** `smapiPin` holds a SMAPI version Mortar installs instead of the latest. The patch-day sweep checks every profile for mods smapi.io marks broken when the game or SMAPI version changes.
- **Install:** the release installer runs unattended (`--install --no-prompt --game-path "<dir>"`); running it again updates. Mortar installs SMAPI itself, without asking, when a profile is created, at startup while SMAPI is missing or broken and before Play after a game update replaced the launcher. Only a newer release waits for the user, since an update can break mods. On Windows Mortar runs the self-contained `SMAPI.Installer.exe` from the extracted zip, as `install on Windows.bat` does (the batch file refuses a `%TEMP%` path).
- **Broken by a game update:** on Linux the installer renames `StardewValley` to `StardewValley-original` and puts its launcher in place, which a game update overwrites. Mortar checks at startup and before launch that `StardewValley-original` exists and `StardewValley` contains `StardewModdingAPI`, else Play reinstalls SMAPI first. On Windows the launcher survives and an incompatible SMAPI reports itself in its log.
- **Bundled mods:** a profile's `mods/` replaces `Mods/`, so every profile gets SMAPI's Console Commands and Save Backup as a `smapi-<version>` store entry from the installer's `install.dat`. A SMAPI update replaces that entry in every profile.
- **Mods path:** SMAPI takes `--mods-path <path>`, absolute or relative to the game folder. Its `SMAPI_MODS_PATH` variable works only when SMAPI starts directly (`steam -applaunch` cannot pass environment variables), so Mortar does not use it. Mortar never moves or writes the game folder's own `Mods`.
- **Scanning:** SMAPI recurses into subfolders until it finds a `manifest.json` and skips folders starting with a dot. `UpdateCautionMessage` and `DeleteOldVersion` (Stardrop update hints) are read with the same case-insensitive key rules; Update all skips cautioned mods until the row is ticked.

### Launch

- **Linux Steam:** `steam -applaunch 413150 --skip-terminal -- --mods-path <absolute path to the profile's mods/>`. SMAPI's launcher script reads its own flags only before `--` and forwards only what follows (unix launcher argument loop in Pathoschild/SMAPI), which is why SMAPI's docs say arguments do not work on Linux. `--skip-terminal` makes the script start SMAPI with `--no-terminal`. Without `--` the path is ignored; without `--skip-terminal` the launcher opens its console in an unreadably small `xterm`.
- **Windows Steam:** Steam starts `Stardew Valley.exe`, not SMAPI, because SMAPI's installer swaps the launcher only on Linux and macOS. First run shows `"<game>\StardewModdingAPI.exe" %command%` to paste into Stardew's Steam launch options; **Set it in Steam for me** writes `LaunchOptions` in the MostRecent account's `localconfig.vdf` while Steam is closed, since Steam rewrites the file on exit (`internal/steam/localconfig.go`). Mortar then adds `--mods-path <path>` after the app ID: SMAPI finds its flags anywhere on its command line and ignores unknown ones. Without the SMAPI line Mortar starts `StardewModdingAPI.exe` itself (`game.steamSkipsLoader`) and loses the Steam overlay.
- **Windows console:** `StardewModdingAPI.exe` is a console program, so Windows gives it a window whatever the flags; Mortar does not pass `--no-terminal` there. A direct launch hides the window (`showSmapiConsole` off); the Console tab follows the log file either way.
- **Console input:** a process Mortar starts gets a stdin pipe it holds open until exit (`launch.startCmd`). SMAPI's console loop skips empty reads, so on `/dev/null` or `NUL` it spins a core for the whole session. Console tab commands go through the bridge mod, not stdin.
- **Startup markers:** SMAPI's `Console.ReadKey` throws when stdin is not a console, so the "game crashed last time" prompt (`smapi-internal/StardewModdingAPI.crash.marker`) ends SMAPI before the game opens and the surviving marker fails every later launch. The SMAPI loader's `Prelaunch` removes that marker and the update marker before each profile launch; `ErrorLogs/SMAPI-crash.txt` is kept.
- **Without mods:** `steam -applaunch 413150` with no arguments, or the game's executable; no profile, lock, `lastPlayed` or run is recorded. On Windows a Steam launch-options line holding SMAPI still starts SMAPI, so Mortar warns first and never edits that file.
- **Success:** `steam -applaunch` returns at once, so a launch counts as started when `SMAPI-latest.txt` is rewritten within 60 s, else failed with a hint.
- **Whose log:** the game has one `SMAPI-latest.txt` for every profile, so the Console and Share log show it only in the profile that wrote it: the session Mortar launched belongs to its profile; the file on disk belongs to the profile whose mods folder matches SMAPI's `Mods go here: <path>` line (SMAPI writes the home folder as `~`); a log without that line belongs to none.
- **Bisect:** a step fails only when SMAPI logs an alert or the run ends crashed; plain errors such as a skipped mod do not count, or the bisect would blame whichever mod logs one.
- Trap: launching through Steam makes Steam Cloud download the user's saves, and SMAPI's Save Backup mod zips them into `save-backups/` in the game folder.

### Mod data

- **Mod dataset:** SMAPI's `Pathoschild/StardewModDataset` (MIT or CC-BY-SA 4.0, published with explicit permission from the sites it lists) records each Nexus mod page's current files with the SMAPI manifests inside, not every file. A file it lacks is checked after download.
- **Update checks:** `POST https://smapi.io/api/v4.0.0/mods` returns `suggestedUpdate`, taken as given; `compatibilityStatus` and `brokeIn` come only with `includeExtendedMetadata: true`. The API is public, unauthenticated and officially unreleased: answers are cached once for every profile, misses go in batches of 100, and after a failed ask Mortar serves stale or unknown answers for a while. Unofficial versions show as Unofficial update available and Update all never installs them.
- **Dependencies:** `Dependencies[]` (`IsRequired` defaults to true) plus `ContentPackFor` as a required dependency on its framework mod. For a missing one Mortar prefers the Nexus page named in its own `UpdateKeys`, else the page whose files hold it at the highest version (one ID can appear on several pages), and takes the newest `MAIN` file satisfying `MinimumVersion`. One found only on CurseForge is listed with its page link.
- **Share log** uploads the SMAPI log to smapi.io (`internal/logshare` over go-smapiweb's `UploadLog`, `POST https://smapi.io/log`, no auth). Trap: the server answers a success with a redirect to `/log/<id>` and a failed or empty upload with HTTP 200 and its page, so only a redirect counts as success. The log holds local paths including the user's name and anyone with the link can read it, so Mortar shows it and asks first. Only a loader with `loader.LogShare` (SMAPI) offers it.

### Content Patcher

Rules: `internal/framework/contentpatcher`. Invariants the code does not say:

- **Conflicts:** two or more enabled packs that `Load` the same target (a Load without `Priority` is `Exclusive`) or that `EditImage`/`EditMap` the same target or `EditData` the same entry or field. `EditData` that only shares a target is not reported. Packs built to work together (one entry, or one names the other as a dependency or in a `HasMod` condition) never overlap.
- **Only edits that can both apply count:** a `HasMod` condition counts only when it holds for the enabled mods; a change whose config conditions do not match the pack's current settings (`config.json`, else schema `Default`) is not counted; edits needing different spouses, or disjoint literal `LocationName`, `LocationContext`, `Season`, `Weather`, `DayOfWeek` or `FarmType` values, never apply together. Image edits conflict only where an opaque 16x16 source cell lands.
- **Make win** records the loser's UniqueID in the winner entry's `loadAfter` and adds it to the winner's installed `manifest.json` as an optional dependency, so SMAPI loads the winner later. The profile's manifest is its own copy, so the store is untouched, and the dependency is rewritten after an update.
- **Asset map:** mods are ranked in SMAPI load order (`internal/loadorder`), not profile order. Trap: for Loads the last loaded wins on a tie although the Content Patcher docs say "first" (`PatchManager.ApplyPatchesToAsset` replaces its pick with every later candidate that is not strictly lower); two Exclusive loads apply neither.
- **Same job** (`internal/problems/samejob.go`, constants `sameJob*`): a C# mod's footprint is the `StardewValley` members its `EntryDll` assigns, read from the IL, plus its Harmony-replaced methods from the startup report. Members are weighted ln(N/df), so a rarely written member counts more.
- **Config suggestions** (`contentpatcher/settings.go`): an on/off `ConfigSchema` field whose value runs none of the pack's `HasMod` compatibility patches is suggested with the values that would; fields that list NPCs never are.

### Generic Mod Config Menu

Menus a mod registers with GMCM or GMCM Options come from mortar-smapi-bridge, which runs inside the game: it writes each menu to `<profile>/gmcm/<UniqueID>.json`. Edits made through `mortar mods menu` wait in `<profile>/gmcm-pending/<UniqueID>.json` until the next launch, when the bridge applies them through the setters registered with GMCM. Widgets the bridge cannot set (colour, image, custom drawing) are read-only.

### Saves

Saves live in one folder, `%APPDATA%\StardewValley\Saves` or `~/.config/StardewValley/Saves`, which Steam Cloud syncs, so Mortar never moves it. Profiles share saves unless one sets `separateSaves` (`internal/savesiso`).

- **Where mods leave traces:** SMAPI writes no mod list into a save, but mods leave keys prefixed with their `UniqueID` (`smapi/mod-data/<uniqueid>/<key>`, `modData` keys such as `Sonozuki.MoreGrass/GrassOffsetX0`). Mortar scans the save's `key`/`string` XML entries and takes for each key the longest prefix ending at a `/`, `_` or `.` boundary that is a `UniqueID` in the mod dataset's index, ignoring case. Splitting at the first `_` would be wrong: many IDs contain `_`. Mods with no per-save data leave no trace and keys outlive a removed mod, so the warning says "this save has used".

## The Sims 4

Each decision names its community source, because EA documents little. The catalog's `folderNames` and `requiredSettings` encode most of it.
- **Language folders:** the user folder follows the game's language and switching starts a new empty one beside the old (EA forums `11781561`; heise.de's Sims 4 mods guide). Mortar uses whichever exists and changed last.
- **Redirected Documents** (OneDrive, a moved folder, a symbolic link) is common (EA forums `11782389`), so paths resolve through the real Documents folder. Under Proton it is `compatdata/1222670/pfx/drive_c/users/steamuser/Documents/Electronic Arts/The Sims 4` (a Steam Deck user's `Mods` path, Steam discussion `4364628453802426195`).
- **Executables:** `Game/Bin` also holds `TS4_DX9_x64.exe`, the DirectX 9 build (EA forums `11819001`).
- **`Options.ini`:** `modsdisabled` must be `0` and `scriptmodsenabled` `1` (EA forums `8908087`). Patches turn the in-game switches off (help.ea.com, "Mods and The Sims 4 game updates") and the game rewrites the file, so script mods can reset to off after a restart (EA forums `13525568`). Hence Mortar keeps the setting in the profile's copy and checks it before Play.
- **Rollback copies:** `Slot_########.save` sits beside `.save.ver0` (newest) to `.ver4` (oldest), which are not saves of their own; removing the `.verN` suffix restores one (EA's backup guide; simscommunity.info).
- **Depth limits:** `Resource.cfg` allows packages five folders below `Mods` (bank42n.com's guide); the game creates it when missing and the stock text is a `Priority 500` block of `PackedFile` lines that CC managers extend. `.ts4script` works only one folder below `Mods` (community guidance; EA gives no figure).
- **Tray files:** the target's `extensions` on role `tray` are placed at install in the shared Tray folder, never swapped at launch, held by an entry of their own (`Entry.Tray`) with each file's SHA-256. A file of the player's own there is never replaced: the install is refused naming it, or an identical file is left unclaimed. Released only while no entry of any profile holds them and the hash still matches; update and roll back place new files before releasing old.
- **Marker:** `GameInfo.MarkerDir` is `Game/Bin`.
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
