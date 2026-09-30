# Mortar architecture

How Mortar works now: storage, profile semantics, trust boundaries, and the Stardew and Nexus facts that settle questions so they are not re-litigated. Current state only; decided work not yet built is in [design.md](design.md), screens in [gui-design.md](gui-design.md), standing rules in [AGENTS.md](../AGENTS.md).

Product decisions (scope, stack, look, sources) are NOMAD's, made on 2026-09-29; implementation details (storage layout, caps, carry-over rules, failure handling) follow from the evidence cited beside them. Every measurement was taken on 2026-09-29 on NOMAD's Fedora (GNOME Wayland, btrfs) machine unless it says otherwise.

## Product

Mortar finds a game, installs its mod loader, keeps each set of mods in its own profile, and launches the game with the chosen profile.

- **What matters:** Mortar working for NOMAD's own use with a personal Nexus API key. Registering the app with Nexus is not a v1 goal.
- **Licence:** AGPL-3.0 (NOMAD, 2026-09-29): the same as GPL-3.0 for the desktop app, and it already covers a hosted share service if one is built here later.
- **Scope of the first release:** Stardew Valley only, on Windows and Linux, from a Steam client installed directly on the system (NOMAD, 2026-09-29: no GOG, Heroic, Lutris or Flatpak Steam in v1). macOS is out (fleet CI has no macOS runners, and Wails does not cross-compile to it).

## Stack

- **Wails v3**, upgraded one beta at a time on purpose; the pin and its Go floor are in `go.mod`. Recent betas are small fixes; there is no date for a stable v3.0.
- **Frontend:** React, TypeScript and MUI on Vite, from Wails' `react` template. Typed bindings from `wails3 generate bindings` into `frontend/bindings`. Client state in zustand. Lingui 6 for every string; English only at first.
- **Go services:** each backend area the UI calls is one Wails service (`application.NewService`), so its exported methods are bound to TypeScript.
- **Solid window:** `BackgroundTypeSolid` (`main.go`); the base, tint and surfaces are in [gui-design.md](gui-design.md#surfaces-and-colour).
- **Frameless, with a themed title bar:** `Frameless: true`, and the app draws its own title bar marked `--wails-draggable: drag`. Measured: drag and edge resizing work; there is no drop shadow on GNOME, so the app draws a 1px border and rounded corners. Trap: dragging works only once the Wails runtime is loaded (`@wailsio/runtime`, or `/wails/runtime.js`).
- **Packages worth knowing:** `internal/game` holds the `Game` interface and `internal/game/stardew` its only implementation; `internal/archive` extracts zip, RAR and 7z (a mod's latest main file is 73.0% zip, 13.1% RAR and 3.1% 7z across the mod dataset, so zip alone would fail about one mod in six).

## Storage

Everything lives in the user data folder, `%LOCALAPPDATA%\Mortar` or `$XDG_DATA_HOME/mortar`, never under the Windows install folder (the uninstaller deletes that one recursively). Mortar's own state is JSON, each file written through a temp file and rename; no database. The Nexus key is in the OS keyring.

- `settings.json`: game folders, the accent colour, the window background and its wallpaper path, the installed SMAPI version, the nxm handler state and the handler it replaced, dismissed save warnings.
- `store/<game>/<key>/`: each downloaded archive, extracted once. Keys: `nexus-<mod id>-<file id>`, `github-<owner>-<repo>-<tag>-<asset>`, `local-<sha256 of the archive>`, `smapi-<version>` for SMAPI's bundled mods, and `bridge-<version>` for the Mortar SMAPI Bridge. One Nexus file can hold several SMAPI mods, so the key is the file.
- `store/index.json`: each store item's last use, meaning the last time any `profile.json` named it.
- `profiles/<game>/<profile id>/profile.json`, and beside it `mods/`, the folder SMAPI is pointed at. Each entry is copied to `mods/<store key>/` exactly as extracted; SMAPI recurses until it finds a `manifest.json`.
- `trash/`: deleted profiles, kept 30 days.
- `backups/`: zips of the Saves folder taken before updates, the last five kept.
- `cache/`: API responses, mod pictures and the dataset index.
- `queue.json`: the download queue's state, without `nxm://` keys.

`profile.json`: `id`, `name`, `notes`, `cover` (a path or empty), `order`, `hidden`, `created`, `updated`, and `entries[]`, each with `key`, `previousKey` (for rollback, or empty), the source reference used in share links, the mods it holds (`UniqueID`, version, the folder holding its `manifest.json`), and `disabled`, the `UniqueID`s switched off. The store keeps every item a profile names as `key` or `previousKey` and deletes the rest 30 days after their last use.

## Profile operations

**Copies, which the filesystem clones where it can.** Each profile gets its own copy of every entry.

- **btrfs:** Go's `io.Copy` between files uses `copy_file_range`, which btrfs turns into a reflink: a 200 MB copy showed 0 exclusive and 200 MB shared bytes (`btrfs filesystem du`).
- **NTFS and ext4:** an ordinary copy, affordable since 100 random mods total a median of 85 MB as archives (90th percentile 485 MB; per mod, median under 0.05 MB, 99th percentile 24 MB, across 33,181 mods).
- **Links are out:** SMAPI's `helper.Data.WriteJsonFile` writes any path inside the mod's folder (`Pathoschild/SMAPI` `src/SMAPI/Framework/ModHelpers/DataHelper.cs:62`) with `File.WriteAllText` (`src/SMAPI.Toolkit/Serialization/JsonHelper.cs`), which rewrites the file in place, so a hard link would leak one profile's data into the store and every other profile. SMAPI does load linked folders (measured).

**Operations:**

- **Toggle** a mod: rename the folder holding its `manifest.json` with a leading dot, which SMAPI skips (measured), and record it in `disabled`. When the manifest sits at the archive's root, that folder is the entry's own `mods/<store key>/`, so the entry is found by its key with or without the dot. Programs can create leading-dot names on NTFS.
- **Duplicate:** copy the profile's `mods/` folder as it is, mod-written data included, under a new id.
- **Delete:** move the profile folder to `trash/`, restorable for 30 days.
- **Update or roll back** an entry: copy the target version from the store into a fresh folder, then carry over what the mod or the user wrote, by comparing three copies of each file: the profile's, the store's copy of the current version, and the target version.
  - A file only the profile's copy has (such as `config.json` or mod data) is carried over.
  - A file the profile changed is carried over when the target version ships it unchanged from the current version. When the target version changed it too, the target's file wins and the profile's copy is kept beside it as `<name>.mortar-old`.
  - Keep the leading dot on disabled folders; then swap `key` and `previousKey` and remove the old folder.
  - Before updating a profile, zip the Saves folder into `backups/`: NOMAD's 803 MB of saves zip to 20 MB in 3.0 s, so five backups cost about 100 MB.

## Trust boundaries

Everything from outside is untrusted: links, `.mortar` files, archives, API responses. Rules for the inputs not yet built are in [design.md](design.md#sharing-milestone-5).

- Extraction keeps every entry inside its destination; caps each entry at 256 MiB, the archive at 2 GiB extracted and 20,000 entries; rejects symlink entries, Windows reserved names (`CON`, `NUL`, ...), `:` in names, and names that differ only by case.
- An `nxm://` link is taken only in the exact form `nxm://stardewvalley/mods/<mod id>/files/<file id>?key=<key>&expires=<unix>&user_id=<id>`, unexpired, and for the signed-in account's `user_id`; anything else is refused with the reason. A link Mortar was not waiting for asks which profile it is for before anything downloads.

## Failure behaviour

Once mods are installed, nothing external is needed to open, edit or launch a profile.

- **SMAPI's API or the dataset unreachable:** the feature degrades and never blocks; update checks show "unknown", dependencies are checked after download.
- **A download cut off or corrupt:** extraction goes to a temp folder on the store's volume and moves into the store only when every entry passed its checksum (CRC32 in zip, RAR and 7z); failure deletes the temp folder and the item can be retried. **Disk full** fails the same way and says how much space the item needs.
- **Mortar quits mid-operation:** every write is temp-then-rename, and startup removes leftover temp folders. A `mods/` folder deleted outside Mortar is rebuilt from the store from `profile.json`.
- **Steam not running:** `-applaunch` starts it. **Not installed:** a Steam copy can launch through SMAPI directly, without the overlay, after the user agrees.

## Tests

Go tests run against local HTTP test servers replaying recorded Nexus, SMAPI API and dataset responses, inside the 10 s warm / 30 s cold gate budget. Recordings made with NOMAD's key are scrubbed before commit: `users/validate.json` returns the account's name and email, so fixtures carry placeholder account fields. An opt-in smoke test copies a real Stardew install to scratch space, installs SMAPI with `--no-prompt`, launches a profile, and reads `SMAPI-latest.txt` for the loaded mods, as the planning tests did.

## Stardew Valley

Sources: `Pathoschild/SMAPI` `docs/technical/smapi.md`, `docs/technical/web.md`, `src/SMAPI.Toolkit/Serialization/Models/Manifest.cs`, `src/SMAPI.Toolkit/Framework/ModScanning/ModScanner.cs` and `src/SMAPI.Installer/assets/unix-launcher.sh`; the Stardew Valley wiki's Modding pages.

### Finding the game

Steam app `413150`, through `libraryfolders.vdf`; a machine can hold several Steam accounts (NOMAD's has two `userdata` folders), so per-account files such as `localconfig.vdf` are read for the account marked `MostRecent` in `config/loginusers.vdf`. Only a directly installed Steam is supported; Mortar says so when it finds only a Flatpak Steam, whose sandbox could not read Mortar's data folder anyway (Flathub's manifest grants no home access). When nothing is found, first run says so and offers Browse and a retry.

### SMAPI

- **Install and update:** download the release installer (4.5.2, published 2026-03-22, needs Stardew 1.6.14+; 2 to 4 releases a year) and run it unattended with `--install --no-prompt --game-path "<dir>"`; running it again updates. Measured on Linux: exit 0, no prompts, launcher replaced, bundled mods added.
  - **When:** Mortar installs SMAPI by itself, without asking (NOMAD, 2026-09-30): in the background when a profile is created or at startup while profiles exist and SMAPI is missing or broken, as first run's second step, and before Play when a game update has replaced the launcher. Only a newer release waits for the user, since an update can break mods.
  - **Windows:** the installer is `SMAPI <version> installer/internal/windows/SMAPI.Installer.exe`, self-contained (its `runtimeconfig.json` bundles .NET 6.0.36). Mortar extracts the whole zip and runs it from the installer folder, as `install on Windows.bat` does (the batch file refuses to run from a `%TEMP%` path).
  - **New releases** come from the GitHub releases client ([GitHub releases](#github-releases)) and show as a banner; one click installs.
- **Versions:** the SMAPI version Mortar installed is kept in `settings.json`; otherwise, and for the game version, Mortar reads the first line of `SMAPI-latest.txt` (`SMAPI 4.5.2 with Stardew Valley 1.6.15 build 24356 on Unix ...`, measured).
- **Broken by a game update:** on Linux the installer renames the game's `StardewValley` to `StardewValley-original` and puts its launcher in its place, which a game update overwrites; Mortar checks at startup and before launch that `StardewValley-original` exists and `StardewValley` contains `StardewModdingAPI`, and Play reinstalls SMAPI first when it is not (the banner keeps a manual Reinstall). On Windows the launcher survives, and an incompatible SMAPI reports itself in its log, which the console surfaces with the same offer.
- **Bundled mods:** a profile's `mods/` replaces `Mods/`, so every profile gets SMAPI's Console Commands and Save Backup as a `smapi-<version>` store entry, taken from the installer Mortar downloaded for the installed version (`internal/<os>/install.dat`, a zip holding `Mods/ConsoleCommands` and `Mods/SaveBackup`, checked in 4.5.2). A SMAPI update replaces that entry in every profile.
- **Mortar SMAPI Bridge:** Mortar has no access to SMAPI's stdin, so a small SMAPI mod (`Rethunk-AI/mortar-smapi-bridge`, `UniqueID` `Rethunk.MortarSmapiBridge`, protocol in its README) accepts console commands on a loopback port.
  - **Install:** the release zip is vendored in `internal/game/stardew/vendor/` (pinned version and sha256 in `bridge.go`, refreshed by `scripts/update-bridge.sh <version>`) and installed hidden in every profile as a `bridge-<version>` store entry, with source kind `mortar`, added at startup and at profile creation and replaced when the version changes. Hidden means as SMAPI's bundled mods are ([gui-design.md](gui-design.md#mods-tab)).
  - **Channel:** the mod writes `mortar-smapi-bridge.json` (`{port, token, pid}`, user-only) in its folder; `internal/bridge` reads it from the running profile, checks the pid, and sends `<token>\n<command>\n`, so the token never leaves the user's own files. Commands run on the game's next tick and their output arrives in SMAPI's log.
- **Profiles folder:** SMAPI takes `--mods-path <path>`, absolute or relative to the game folder (`smapi.md:49`). Its `SMAPI_MODS_PATH` environment variable works only when SMAPI is started directly, since `steam -applaunch` cannot pass environment variables, so Mortar does not use it. The game folder's own `Mods` is never read, moved or changed in v1.
- **Scanning:** SMAPI recurses into subfolders until it finds a `manifest.json` and skips folders whose names start with a dot. Manifests are parsed leniently.

### Launch

Launch goes through `steam -applaunch <appid> <args>`: Steam passes the arguments after the app ID to the game, which keeps the overlay, achievements and playtime tracking.

- **Linux:** `steam -applaunch 413150 --skip-terminal -- --mods-path <absolute path to the profile's mods/>`. SMAPI's launcher script reads its own flags only before `--` and forwards only what follows it (`Pathoschild/SMAPI` `src/SMAPI.Installer/assets/unix-launcher.sh`, the argument loop), which is why SMAPI's docs say arguments do not work on Linux. `--skip-terminal` makes the script start SMAPI with `--no-terminal`; SMAPI still writes its log file then (`docs/technical/smapi.md:46`). Measured: without `--` the path is ignored; with it SMAPI loads the profile, including through NOMAD's real Steam client, where the game reached the main menu. Without `--skip-terminal` the launcher opened its console in `xterm`, unreadably small.
- **Windows:** Steam starts `Stardew Valley.exe`, not SMAPI, so first run shows the line `"<game>\StardewModdingAPI.exe" %command%` to paste into Stardew's Steam launch options once, with a copy button; Mortar reads `LaunchOptions` from `localconfig.vdf` to warn when it is missing, and never writes that file (Steam rewrites it while running). Mortar then adds `--mods-path <path>` after the app ID.
- **Success or failure:** `steam -applaunch` returns at once, so a launch counts as started when `SMAPI-latest.txt` (`~/.config/StardewValley/ErrorLogs/`, `%APPDATA%\StardewValley\ErrorLogs\`) is rewritten within 60 s, and failed otherwise, with a hint (on Windows, the missing launch-options line). When SMAPI exits having written no log, the console shows its exit code.
- **Running:** a `StardewModdingAPI` process whose `--mods-path` is a profile's folder means that profile is in use; its folder is locked against changes until the process exits (Windows locks loaded files anyway).
- Trap: launching through Steam makes Steam Cloud download the user's saves, and SMAPI's Save Backup mod zips them into `save-backups/` in the game folder.

### Saves

Saves live in one folder, `%APPDATA%\StardewValley\Saves` or `~/.config/StardewValley/Saves`, which Steam Cloud syncs, so Mortar never moves it; profiles share saves. Each profile's Saves card ([gui-design.md](gui-design.md#main-screen)) lists every save with the mods it has used that the profile lacks, read from the save itself, as NOMAD recalled a mod once doing in game. Players pick their save inside the game, so there is no per-launch check.

- **Where mods leave traces:** SMAPI writes no mod list into a save, but mods leave keys prefixed with their `UniqueID`: SMAPI's save data `smapi/mod-data/<uniqueid>/<key>`, lowercased whole (`Pathoschild/SMAPI` `src/SMAPI/Framework/ModHelpers/DataHelper.cs:148`), and `modData` keys such as `Sonozuki.MoreGrass/GrassOffsetX0`.
- **Matching:** Mortar scans the save's `<key><string>...</string></key>` entries and, ignoring case, takes for each key the longest prefix ending at a `/`, `_` or `.` boundary that is a `UniqueID` in the mod dataset's index. Splitting at the first `_` would be wrong: 1,413 IDs contain `_`, and for 81 of them the part before it is another mod's ID; 1,081 have no dot.
- **Cost:** measured on one of NOMAD's saves: 65 MB, 330,748 keys, 32 mods in 0.43 s. Results are cached by the save file's modification time.
- **Limits:** mods that keep no per-save data leave no trace, and keys outlive a mod removed on purpose, so the warning says "this save has used", and each mod can be dismissed for that save.

### Mod data

- **Mod dataset:** SMAPI's `Pathoschild/StardewModDataset` (MIT or CC-BY-SA 4.0, published with explicit permission from Nexus Mods, CurseForge and ModDrop, refreshed near the end of each month, 0.x with no version tags) records each Nexus mod page's current files with the SMAPI manifests inside them (`Downloads[].Mods[]`, `SizeInBytes`), not every file: Content Patcher has 2 there against 170 on Nexus.
  - **Fetching:** one page's entry when needed, from `https://raw.githubusercontent.com/Pathoschild/StardewModDataset/main/dataset/data/Nexus/<mod id / 1000>/<mod id>.json`; the `UniqueID` index (`dataset/indexes/pages by mod ID.json`, 1.5 MB, 28,634 IDs) monthly.
  - **Parsing** is lenient; a file it lacks is checked after download.
- **Update checks:** `POST https://smapi.io/api/v4.0.0/mods` with each mod's ID, `UpdateKeys` and installed version, plus `apiVersion`, `gameVersion` and `platform`, returns `suggestedUpdate`, which Mortar takes as given; `compatibilityStatus` and `brokeIn` come only with `includeExtendedMetadata: true`. The API is public, unauthenticated and officially unreleased, so results are cached and failures never block. Updates show as a count per profile and are applied only by the user.
- **Dependencies:** `Dependencies[]` (`UniqueID`, `MinimumVersion`, `IsRequired` defaulting to true) plus `ContentPackFor` as a required dependency on its framework mod, all by `UniqueID`. For a missing one, Mortar prefers the Nexus page named in its own `UpdateKeys`, else the page whose files hold it at the highest version (3,793 IDs appear on several pages), and takes the newest `MAIN` file satisfying `MinimumVersion`. One found only on CurseForge or ModDrop is listed with its page link, to install from a downloaded archive.
- **Problems** warn and never block: missing dependencies, duplicate `UniqueID`s (resolved in a dialog that keeps one copy and switches the other off), and mods SMAPI's API marks broken for the game version, each with a one-click fix.

## Nexus Mods

Sources: the API acceptable-use policy (help.nexusmods.com article 114), the SSO demo `Nexus-Mods/sso-integration-demo`, the `Nexus-Mods/node-nexus-api` client, and Vortex's (`Nexus-Mods/Vortex`) `NXMUrl.ts`.

- **Sign-in:** Settings takes a personal API key, kept in the OS keyring and never sent to a server of ours. `GET /v1/users/validate.json` gives the account's `user_id` and `is_premium`. Every request carries a truthful `Application-Name` and `Application-Version`.
- **Rate limits** (measured on NOMAD's free account): `x-rl-daily-limit: 20000`, `x-rl-hourly-limit: 2000`, with `-remaining` and `-reset` headers; the client reads them on every response and pauses before running out.
- **Calls only on the user's action:** a mod's `picture_url` and `endorsement_count` (`GET /v1/games/stardewvalley/mods/<id>.json`) are fetched once, when it is installed, and cached.
- **Files:** `GET /v1/games/stardewvalley/mods/<id>/files.json` lists every file (170 for Content Patcher) with `file_id`, `file_name`, `version`, `mod_version`, `category_name` (`MAIN`, `OPTIONAL`, `UPDATE`, `MISCELLANEOUS`, `OLD_VERSION`, `ARCHIVED`, or null), `size_kb` and `is_primary`. An update takes the `MAIN` file matching SMAPI's `suggestedUpdate`, else the `is_primary` one, and never an `OPTIONAL` file unless the profile already uses it.
- **Downloads:** `GET /v1/games/stardewvalley/mods/<id>/files/<file id>/download_link.json`. Premium accounts get links directly; they expire, so Mortar asks just before downloading and again on a restart. A free account gets HTTP 403 ("...this is for premium users only", measured); it needs the `key` and `expires` from the `nxm://` link that the site's Mod Manager Download produces, one click per mod.
- **`nxm://` links:** `nxm://<game>/mods/<mod id>/files/<file id>?key=<key>&expires=<unix>&user_id=<id>`; Mortar takes this form only. Only one app can own the scheme, and Wails' NSIS macro would delete and rewrite its key at install time without asking, and delete it on uninstall (`build/windows/nsis/wails_tools.nsh`, from Wails' template), taking it from Vortex or Mod Organizer 2. So `nxm` is not in `build/config.yml`: Mortar registers it at runtime after the user agrees (`HKCU\Software\Classes\nxm` on Windows; on Linux a user-level `~/.local/share/applications/mortar.desktop` lists `x-scheme-handler/nxm` and `xdg-mime default` is set; turning it off drops the type again), records the previous handler in `settings.json`, and restores it when turned off.
- **No re-hosting:** mod files and Nexus data stay on the user's machine; a share link holds IDs only.
- **Download queue:** an import or a profile's missing files run one queue (`internal/queue`), persisted in `<datadir>/queue.json` without the `nxm://` keys. Items store as `nexus-<mod id>-<file id>`, and the entry records source kind `nexus` with the mod id, file id, version, picture URL and endorsement count, fetched once at install. Premium accounts download without clicks. A free account's head item waits for its click, and a matching `nxm://` link is routed to it instead of showing the arrival card. A reached rate limit pauses the queue until Nexus's reset time.

## GitHub releases

`internal/github` reads `/repos/<owner>/<repo>/releases` without sign-in. Answers are cached on disk for an hour and served stale when GitHub is unreachable. The unauthenticated limit is 60 calls an hour per IP, and a reached limit shows its retry time. An install or update takes the release's only archive asset, or asks when there are several; the download is capped at the archive extraction limit and stores as `github-<owner>-<repo>-<tag>-<asset>`. SMAPI's own release lookup uses the same client.

**Trust:** a GitHub-sourced install counts as verified only when SMAPI's update API (`includeExtendedMetadata: true`) maps the manifest's `UniqueID` to the same repo. It is unknown when the API is unreachable, and never blocks.
