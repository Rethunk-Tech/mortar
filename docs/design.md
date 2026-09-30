# Mortar: decided, not yet built

This file holds only work that is decided but not built. Each item is written to be implemented cold: the shape, the evidence it rests on, the traps, and when it is done. When an item lands, delete it here; the code is the record of what exists. Screens and styling: [gui-design.md](gui-design.md). Lethal Company research for the second game: [lethal-company.md](lethal-company.md). Standing rules: [AGENTS.md](../AGENTS.md).

Nothing is built yet. Implementation starts only on NOMAD's explicit go-ahead. Items marked **Measure** need a throwaway test first; those tests run outside this repo and only their results land here. Product decisions (scope, stack, look, sources, sharing, what is deferred) are NOMAD's, made on 2026-09-29; implementation details (storage layout, caps, carry-over rules, failure handling) were proposed during planning, reviewed in three audit passes, and follow from the evidence cited beside them. Every measurement was taken on 2026-09-29 on NOMAD's Fedora (GNOME Wayland, btrfs) machine unless it says otherwise.

## Product

Mortar finds a game, installs its mod loader, keeps each set of mods in its own profile, and launches the game with the chosen profile. A profile is shared as a link, and opening the link on another machine produces the same mods, with every dependency checked before anything downloads.

- **What matters:** Mortar working for NOMAD's own use with a personal Nexus API key. Registering the app with Nexus is not a v1 goal.
- **Licence:** AGPL-3.0 (NOMAD, 2026-09-29): the same as GPL-3.0 for the desktop app, and it already covers a hosted share service if one is built here later.
- **Scope of the first release:** Stardew Valley only, on Windows and Linux, from a Steam client installed directly on the system (NOMAD, 2026-09-29: no GOG, Heroic, Lutris or Flatpak Steam in v1). macOS is out (fleet CI has no macOS runners, and Wails does not cross-compile to it). Everything deferred is listed under Later.

## Stack

- **Wails v3** pinned to `v3.0.0-beta.26` (needs Go 1.25+, `v3/go.mod:3`), upgraded one beta at a time on purpose. Recent betas are small fixes; there is no date for a stable v3.0.
- **Frontend:** React, TypeScript and MUI on Vite, from Wails' `react` template (`wails3 init -t react`; `react-js` is the plain JavaScript one). Typed bindings from `wails3 generate bindings` into `frontend/bindings`. Client state in zustand. Lingui 6 for every string, wired as Maître's GUI does (`@lingui/vite-plugin` with the Babel macro preset, `Rethunk-AI/maitre` `apps/gui/frontend/vite.config.ts:1`); English only at first.
- **Go services:** each backend area the UI calls is one Wails service (`application.NewService`, optional `ServiceStartup`/`ServiceShutdown`), so its exported methods are bound to TypeScript.
- **`mortar://`** goes in `protocols:` in `build/config.yml`, which the NSIS installer registers on Windows (`nxm://` does not; see Nexus Mods).
- **Single instance:** `Options.SingleInstance` with `OnSecondInstanceLaunch`. A link reaches the app as `events.Common.ApplicationLaunchedWithUrl` on a cold start (fired when the only argument contains `://`, `application_linux.go:73-83`) or in `SecondInstanceData.Args` when it is already running. Both feed one link router.
- **New Go dependencies** (licences checked, all maintained): `github.com/andybalholm/brotli` (MIT; the standard library has no Brotli), `github.com/nwaples/rardecode` (BSD-2-Clause) and `github.com/bodgit/sevenzip` (BSD-3-Clause) for RAR and 7z, `github.com/tailscale/hujson` (BSD-3-Clause; SMAPI manifests are JSON with comments and trailing commas), `github.com/andygrunwald/vdf` (MIT; Steam's `libraryfolders.vdf`, as Concrete used), `github.com/zalando/go-keyring` (MIT; already a Wails dependency, and Maître's).
- **Code from Concrete** (NOMAD's): `src/app/steam/` becomes `internal/steam`, and `secureArchivePath` and `extractEntry` (`src/app/launcher/DoLaunchGame.go`) seed `internal/archive`, under Mortar's AGPL-3.0.

## Look

- **Solid window:** `BackgroundTypeSolid`, base `rgb(25,25,30)`. The wallpaper backdrop sits under a static `rgba(25,25,30,0.8)` tint, and the theme's 80% surfaces composite over it; the rest of the theme is in gui-design.md.
- **Frameless, with a themed title bar:** `Frameless: true`, and the app draws its own title bar marked `--wails-draggable: drag`. Measured: drag and edge resizing work; there is no drop shadow on GNOME, so the app draws a 1px border and rounded corners. Trap: dragging works only once the Wails runtime is loaded (`@wailsio/runtime`, or `/wails/runtime.js`). **Measure:** frameless under KDE, and Acrylic with a frameless window on Windows.

## Architecture

### Layout

One Go module and one Vite frontend, no workspaces:

- `main.go`: services, single instance, link router, updater, window options.
- `internal/game`: the `Game` interface (discover, loader, sources, identity, launch) and shared types; `internal/game/stardew` is the only implementation in v1.
- `internal/steam`: Steam and library discovery.
- `internal/archive`: safe extraction of zip, RAR and 7z. Mods' latest main files are 73.0% zip, 13.1% RAR and 3.1% 7z across the mod dataset, so zip alone would fail about one mod in six.
- `internal/store`: downloads, verification, extraction, and copying into profiles.
- `internal/profile`: `profile.json`, toggles, duplicate, delete, and the save scan.
- `internal/source/nexus`, `internal/source/github`: API clients and the download queue.
- `internal/meta`: SMAPI's update API, the mod dataset, and their caches.
- `internal/share`: link and `.mortar` encode and decode.
- `internal/launch`: Steam launch, launch detection, and the SMAPI log tail sent to the frontend as events.

### Storage

Everything lives in the user data folder, `%LOCALAPPDATA%\Mortar` or `$XDG_DATA_HOME/mortar`, never under the Windows install folder (the uninstaller deletes that one recursively). Mortar's own state is JSON, each file written through a temp file and rename; no database. The Nexus key is in the OS keyring.

- `settings.json`: game folders, the accent colour, the window background and its wallpaper path, the installed SMAPI version, the nxm handler state and the handler it replaced, dismissed save warnings.
- `store/<game>/<key>/`: each downloaded archive, extracted once. Keys: `nexus-<mod id>-<file id>`, `github-<owner>-<repo>-<tag>-<asset>`, `local-<sha256 of the archive>`, and `smapi-<version>` for SMAPI's bundled mods. One Nexus file can hold several SMAPI mods, so the key is the file.
- `store/index.json`: each store item's last use, meaning the last time any `profile.json` named it.
- `profiles/<game>/<profile id>/profile.json`, and beside it `mods/`, the folder SMAPI is pointed at. Each entry is copied to `mods/<store key>/` exactly as extracted; SMAPI recurses until it finds a `manifest.json`.
- `trash/`: deleted profiles, kept 30 days.
- `backups/`: zips of the Saves folder taken before updates, the last five kept.
- `cache/`: API responses, mod pictures and the dataset index.

`profile.json`: `id`, `name`, `notes`, `cover` (a path or empty), `order`, `hidden`, `created`, `updated`, and `entries[]`, each with `key`, `previousKey` (for rollback, or empty), the source reference used in share links, the mods it holds (`UniqueID`, version, the folder holding its `manifest.json`), and `disabled`, the `UniqueID`s switched off. The store keeps every item a profile names as `key` or `previousKey` and deletes the rest 30 days after their last use.

**Copies, which the filesystem clones where it can.** Each profile gets its own copy of every entry. Go's `io.Copy` between files uses `copy_file_range`, which btrfs turns into a reflink: a 200 MB copy showed 0 exclusive and 200 MB shared bytes (`btrfs filesystem du`). On NTFS and ext4 it is an ordinary copy, affordable since 100 random mods total a median of 85 MB as archives (90th percentile 485 MB; per mod, median under 0.05 MB, 99th percentile 24 MB, across 33,181 mods). Links are out: SMAPI's `helper.Data.WriteJsonFile` writes any path inside the mod's folder (`src/SMAPI/Framework/ModHelpers/DataHelper.cs:62`) with `File.WriteAllText` (`src/SMAPI.Toolkit/Serialization/JsonHelper.cs:150`), which rewrites the file in place, so a hard link would leak one profile's data into the store and every other profile. (SMAPI does load linked folders; measured.)

**Profile operations:**

- **Toggle** a mod: rename the folder holding its `manifest.json` with a leading dot, which SMAPI skips (measured), and record it in `disabled`. When the manifest sits at the archive's root, that folder is the entry's own `mods/<store key>/`, so the entry is found by its key with or without the dot. Programs can create leading-dot names on NTFS.
- **Duplicate:** copy the profile's `mods/` folder as it is, mod-written data included, under a new id.
- **Delete:** move the profile folder to `trash/`, restorable for 30 days; the confirm dialog says so.
- **Update or roll back** an entry: copy the target version from the store into a fresh folder, then carry over what the mod or the user wrote, by comparing three copies of each file: the profile's, the store's copy of the current version, and the target version. A file only the profile's copy has (such as `config.json` or mod data) is carried over. A file the profile changed is carried over when the target version ships it unchanged from the current version; when the target version changed it too, the target's file wins and the profile's copy is kept beside it as `<name>.mortar-old`. Keep the leading dot on disabled folders; then swap `key` and `previousKey` and remove the old folder. Before updating a profile, zip the Saves folder into `backups/`: NOMAD's 803 MB of saves zip to 20 MB in 3.0 s, so five backups cost about 100 MB.

### Trust boundaries

Everything from outside is untrusted: links, `.mortar` files, archives, API responses.

- Link parsing accepts only the known forms, caps the encoded payload at 8 KB and the decompressed payload at 64 KB before parsing.
- Extraction keeps every entry inside its destination; caps each entry at 256 MiB, the archive at 2 GiB extracted and 20,000 entries; rejects symlink entries, Windows reserved names (`CON`, `NUL`, ...), `:` in names, and names that differ only by case.
- Nothing downloads or launches from a link by itself: every import shows a preview and waits.
- An `nxm://` link is taken only for game domain `stardewvalley` and the signed-in `user_id`; it completes a waiting queue item, and an unexpected one asks which profile it is for first.
- A GitHub reference in a link is marked unverified in the preview. After download, the mod installs only if SMAPI's update API (`includeExtendedMetadata: true`) gives the manifest's `UniqueID` the same `metadata.gitHubRepo`; otherwise the user sees the mismatch and decides.
- A `.mortar` file writes only `.json` files inside the mod folders its profile installs.
- The clipboard is read only when the user presses Import.

## Stardew Valley

Sources: SMAPI's `docs/technical/smapi.md`, `docs/technical/web.md`, `Manifest.cs`, `ModScanner.cs` and `unix-launcher.sh`; the Stardew Valley wiki's Modding pages.

### Finding the game

- Steam app `413150`, through `libraryfolders.vdf`; a machine can hold several Steam accounts (NOMAD's has two `userdata` folders), so per-account files such as `localconfig.vdf` are read for the account marked `MostRecent` in `config/loginusers.vdf`. Only a directly installed Steam is supported; Mortar says so when it finds only a Flatpak Steam, whose sandbox could not read Mortar's data folder anyway (Flathub's manifest grants no home access). When nothing is found, first run says so and offers Browse and a retry.

### SMAPI

- **Install and update:** download the release installer (4.5.2, published 2026-03-22, needs Stardew 1.6.14+; 2 to 4 releases a year) and run it unattended with `--install --no-prompt --game-path "<dir>"`; running it again updates. Measured on Linux: exit 0, no prompts, launcher replaced, bundled mods added. The Windows installer is `SMAPI <version> installer/internal/windows/SMAPI.Installer.exe`, self-contained (its `runtimeconfig.json` bundles .NET 6.0.36); Mortar extracts the whole zip and runs it from the installer folder, as `install on Windows.bat` does (the batch file refuses to run from a `%TEMP%` path). New SMAPI releases come from GitHub's releases API and show as a banner; one click installs.
- **Versions:** the SMAPI version Mortar installed is kept in `settings.json`; otherwise, and for the game version, Mortar reads the first line of `SMAPI-latest.txt` (`SMAPI 4.5.2 with Stardew Valley 1.6.15 build 24356 on Unix ...`, measured).
- **Broken by a game update:** on Linux the installer renames the game's `StardewValley` to `StardewValley-original` and puts its launcher in its place, which a game update overwrites; Mortar checks at startup and before launch that `StardewValley-original` exists and `StardewValley` contains `StardewModdingAPI`, and offers the reinstall. On Windows the launcher survives, and an incompatible SMAPI reports itself in its log, which the console surfaces with the same offer.
- **Bundled mods:** a profile's `mods/` replaces `Mods/`, so every profile gets SMAPI's Console Commands and Save Backup as a `smapi-<version>` store entry, taken from the installer Mortar downloaded for the installed version (`internal/<os>/install.dat`, a zip holding `Mods/ConsoleCommands` and `Mods/SaveBackup`, checked in 4.5.2). A SMAPI update replaces that entry in every profile.
- **Profiles folder:** SMAPI takes `--mods-path <path>`, absolute or relative to the game folder (`smapi.md:49`). Its `SMAPI_MODS_PATH` environment variable works only when SMAPI is started directly, since `steam -applaunch` cannot pass environment variables, so Mortar does not use it. The game folder's own `Mods` is never read, moved or changed in v1.
- **Scanning:** SMAPI recurses into subfolders until it finds a `manifest.json` and skips folders whose names start with a dot. Manifests are parsed leniently.

### Launch

Concrete launched with `steam -applaunch <appid> <args>`, and Steam passes the arguments after the app ID to the game, which keeps the overlay, achievements and playtime tracking.

- **Linux:** `steam -applaunch 413150 --skip-terminal -- --mods-path <absolute path to the profile's mods/>`. SMAPI's launcher script reads its own flags (`--skip-terminal`, which then starts SMAPI with `--no-terminal`; SMAPI still writes its log file then, `smapi.md:46`) only before `--` and forwards only what follows it (`unix-launcher.sh:35-43`), which is why SMAPI's docs say arguments do not work on Linux. Measured: without `--` the path is ignored; with it SMAPI loads the profile, including through NOMAD's real Steam client, where the game reached the main menu. Without `--skip-terminal` the launcher opened its console in `xterm`, unreadably small.
- **Windows:** Steam starts `Stardew Valley.exe`, not SMAPI, so first run shows the line `"<game>\StardewModdingAPI.exe" %command%` to paste into Stardew's Steam launch options once, with a copy button; Mortar reads `LaunchOptions` from `localconfig.vdf` to warn when it is missing, and never writes that file (Steam rewrites it while running). Mortar then adds `--mods-path <path>` after the app ID. **Measure on Windows:** how the arguments order around `%command%`, and whether SMAPI needs `--no-terminal`.
- **Success or failure:** `steam -applaunch` returns at once, so a launch counts as started when `SMAPI-latest.txt` (`~/.config/StardewValley/ErrorLogs/`, `%APPDATA%\StardewValley\ErrorLogs\`) is rewritten within 60 s, and failed otherwise, with a hint (on Windows, the missing launch-options line). When SMAPI exits having written no log, the console shows its exit code.
- **Running:** a `StardewModdingAPI` process whose `--mods-path` is a profile's folder means that profile is in use; its folder is locked against changes until the process exits (Windows locks loaded files anyway).
- **Console:** the Console tab tails `SMAPI-latest.txt` from the moment the launch counts as started.
- Trap: launching through Steam makes Steam Cloud download the user's saves, and SMAPI's Save Backup mod zips them into `save-backups/` in the game folder.

### Saves

Saves live in one folder, `%APPDATA%\StardewValley\Saves` or `~/.config/StardewValley/Saves`, which Steam Cloud syncs, so Mortar never moves it; profiles share saves. Each profile's Saves card (gui-design.md) lists every save with the mods it has used that the profile lacks, read from the save itself, as NOMAD recalled a mod once doing in game; players pick their save inside the game, so there is no per-launch check. SMAPI writes no mod list into a save, but mods leave keys prefixed with their `UniqueID`: SMAPI's save data `smapi/mod-data/<uniqueid>/<key>`, lowercased whole (`DataHelper.cs:148`), and `modData` keys such as `Sonozuki.MoreGrass/GrassOffsetX0`. Mortar scans the save's `<key><string>...</string></key>` entries and, ignoring case, takes for each key the longest prefix ending at a `/`, `_` or `.` boundary that is a `UniqueID` in the mod dataset's index (splitting at the first `_` would be wrong: 1,413 IDs contain `_`, and for 81 of them the part before it is another mod's ID; 1,081 have no dot). Measured on one of NOMAD's saves: 65 MB, 330,748 keys, 32 mods in 0.43 s. Mods that keep no per-save data leave no trace, and keys outlive a mod removed on purpose, so the warning says "this save has used", and each mod can be dismissed for that save. Results are cached by the save file's modification time.

### Mod data

- **Mod dataset:** SMAPI's `Pathoschild/StardewModDataset` (MIT or CC-BY-SA 4.0, published with explicit permission from Nexus Mods, CurseForge and ModDrop, refreshed near the end of each month, 0.x with no version tags) records each Nexus mod page's current files with the SMAPI manifests inside them (`Downloads[].Mods[]`, `SizeInBytes`), not every file: Content Patcher has 2 there against 170 on Nexus. Mortar fetches one page's entry when needed from `https://raw.githubusercontent.com/Pathoschild/StardewModDataset/main/dataset/data/Nexus/<mod id / 1000>/<mod id>.json`, and the `UniqueID` index (`dataset/indexes/pages by mod ID.json`, 1.5 MB, 28,634 IDs) monthly. Parsing is lenient; a file it lacks is checked after download.
- **Update checks:** `POST https://smapi.io/api/v4.0.0/mods` with each mod's ID, `UpdateKeys` and installed version, plus `apiVersion`, `gameVersion` and `platform`, returns `suggestedUpdate`, which Mortar takes as given; `compatibilityStatus` and `brokeIn` come only with `includeExtendedMetadata: true`. The API is public, unauthenticated and officially unreleased, so results are cached and failures never block. Updates show as a count per profile and are applied only by the user.
- **Dependencies:** `Dependencies[]` (`UniqueID`, `MinimumVersion`, `IsRequired` defaulting to true) plus `ContentPackFor` as a required dependency on its framework mod, all by `UniqueID`. For a missing one, Mortar prefers the Nexus page named in its own `UpdateKeys`, else the page whose files hold it at the highest version (3,793 IDs appear on several pages), and takes the newest `MAIN` file satisfying `MinimumVersion`. One found only on CurseForge or ModDrop is listed with its page link, to install from a downloaded archive.
- **Problems** warn and never block: missing dependencies, duplicate `UniqueID`s (resolved in a dialog that keeps one copy and switches the other off), and mods SMAPI's API marks broken for the game version, each with a one-click fix.

## Nexus Mods

Sources: the API acceptable-use policy (help.nexusmods.com article 114), the SSO demo `Nexus-Mods/sso-integration-demo`, the `Nexus-Mods/node-nexus-api` client, and Vortex's `NXMUrl.ts`.

- **Sign-in:** Settings takes a personal API key, kept in the OS keyring and never sent to a server of ours. `GET /v1/users/validate.json` gives the account's `user_id` and `is_premium`. Every request carries a truthful `Application-Name` and `Application-Version`.
- **Rate limits** (measured on NOMAD's free account): `x-rl-daily-limit: 20000`, `x-rl-hourly-limit: 2000`, with `-remaining` and `-reset` headers; the client reads them on every response and pauses before running out.
- **Calls only on the user's action:** a mod's `picture_url` and `endorsement_count` (`GET /v1/games/stardewvalley/mods/<id>.json`) are fetched once, when it is installed, and cached.
- **Files:** `GET /v1/games/stardewvalley/mods/<id>/files.json` lists every file (170 for Content Patcher) with `file_id`, `file_name`, `version`, `mod_version`, `category_name` (`MAIN`, `OPTIONAL`, `UPDATE`, `MISCELLANEOUS`, `OLD_VERSION`, `ARCHIVED`, or null), `size_kb` and `is_primary`. A share link names the exact file. An update takes the `MAIN` file matching SMAPI's `suggestedUpdate`, else the `is_primary` one, and never an `OPTIONAL` file unless the profile already uses it.
- **Downloads:** `GET /v1/games/stardewvalley/mods/<id>/files/<file id>/download_link.json`. Premium accounts get links directly; they expire, so Mortar asks just before downloading and again on a restart. A free account gets HTTP 403 ("...this is for premium users only", measured); it needs the `key` and `expires` from the `nxm://` link that the site's Mod Manager Download produces, one click per mod. So an import runs a guided queue: Mortar opens each missing file's download page in turn, `https://www.nexusmods.com/stardewvalley/mods/<id>?tab=files&file_id=<file id>&nmm=1`, the URL Vortex opens for free accounts (`src/renderer/src/extensions/nexus_integration/eventHandlers.ts:537-539`), takes the `nxm://` link from the user's click, downloads and installs it, and opens the next. An expired key reopens its page.
- **`nxm://` links:** `nxm://<game>/mods/<mod id>/files/<file id>?key=<key>&expires=<unix>&user_id=<id>`; Mortar takes this form only. Only one app can own the scheme, and Wails' NSIS macro would delete and rewrite its key at install time without asking, and delete it on uninstall (`wails_tools.nsh.tmpl:245-262`), taking it from Vortex or Mod Organizer 2. So `nxm` is not in `build/config.yml`: Mortar registers it at runtime after the user agrees (`HKCU\Software\Classes\nxm` on Windows; on Linux its `.desktop` file gains `x-scheme-handler/nxm` and `xdg-mime default` is set), records the previous handler in `settings.json`, and restores it when turned off.
- **No re-hosting:** mod files and Nexus data stay on the user's machine; a share link holds IDs only.

## GitHub releases

An `UpdateKeys` entry `GitHub:<owner>/<repo>` maps to `/repos/<owner>/<repo>/releases`, read without sign-in (60 calls an hour per IP, so results are cached, update checks go through SMAPI's API first, and a reached limit shows its retry time). An install or update takes the release's only archive asset, or asks when there are several.

## Share links

- **Form:** Discord makes only `http://`, `https://` and `discord://` links clickable (Discord community posts 14574789338135 and 25441082215447), so a shared link is `https://mortar.rethunk.tech/stardew/p#<payload>`. The page is a static file, `site/stardew/p/index.html` in this repo, deployed as a DigitalOcean App Platform static site (free tier: three static apps); `rethunk.tech` is on DigitalOcean's nameservers, and `maitre.rethunk.tech` is already a CNAME to an App Platform app, so the subdomain is set up the same way. Browsers never send the part after `#`, so no profile data reaches any server. The page shows two buttons, since it cannot tell whether a scheme handler exists: open in Mortar (`mortar://stardew/p/<payload>`) and download Mortar, which also copies the link so the importer can take it after installing. The app accepts either form pasted into its import box.
- **Payload:** compact JSON `[1, "<profile name>", [<entry>, ...]]`, the leading number being the format version (an older Mortar meeting a newer one says to update), Brotli-compressed at quality 11 and base64url-encoded without padding. An entry is `[<nexus mod id>, <nexus file id>]`, or `"<owner>/<repo>@<tag>/<asset>"` for GitHub. Only enabled entries with a source are shared; entries from local archives are listed at Share as left out. Measured on real mod IDs sampled from the mod dataset: the whole link is 498 characters for 50 mods, 879 for 100 and 1,630 for 200, so about 240 mods fit a 2,000-character Discord message. `UniqueID`s would push 100 mods to 3,435 characters, since they barely compress, so the link has none. Brotli beats the standard library's deflate by 12 to 15% at every size.
- **Import:** the preview matches entries the user already has by Nexus mod and file ID (from `profile.json`), resolves the rest through the mod dataset to `UniqueID`s and dependencies, groups mods as installed, to download, dependencies added, checked after download (files the dataset lacks), and unavailable (with the page link); importing installs everything available and leaves the unavailable ones listed on the profile. A shared file Nexus has since deleted or archived is replaced by the `MAIN` file with the same version, else the primary file, marked "different file".
- **`.mortar` file:** a zip holding `profile.json` (the same entries as a link) and `configs/<UniqueID>/<relative path>.json` for each enabled mod's `.json` files. Share always offers it as "with settings" and suggests it over about 240 mods. It opens by drop, file picker, or double-click (a `.mortar` file association registered by the installer on Windows and by the `.desktop` file on Linux).

## Support

The Console tab's **Get help** uploads the current SMAPI log to smapi.io/log, the Stardew community's standard support tool, and copies the link; a second button opens a prefilled GitHub issue for Mortar's own bugs once the repo is public. The upload is `POST https://smapi.io/log` with a form-encoded body `input=<log text>`, no auth; the server answers with a redirect to `/log/<id>`, which is the link, and answers a failed or empty upload with HTTP 200 and its page, so success means a redirect and nothing else (`src/SMAPI.Web/Controllers/LogParserController.cs:119-150`). A SMAPI log holds local paths, including the user's name, and anyone with the link can read it, so Mortar shows the log and asks first.

## Failure behaviour

Once mods are installed, nothing external is needed to open, edit or launch a profile.

- **Nexus unreachable or erroring:** the queue pauses and retries with backoff. **Rate limit reached:** it waits for the reset time and shows it.
- **SMAPI's API, the dataset or GitHub unreachable:** the feature degrades and never blocks; update checks show "unknown", dependencies are checked after download.
- **A download cut off or corrupt:** extraction goes to a temp folder on the store's volume and moves into the store only when every entry passed its checksum (CRC32 in zip, RAR and 7z); failure deletes the temp folder and the item can be retried. **Disk full** fails the same way and says how much space the item needs.
- **Mortar quits mid-operation:** every write is temp-then-rename, and startup removes leftover temp folders. A `mods/` folder deleted outside Mortar is rebuilt from the store from `profile.json`.
- **Steam not running:** `-applaunch` starts it. **Not installed:** a Steam copy can launch through SMAPI directly, without the overlay, after the user agrees.

## Releases

- **Packaging:** an AppImage on Linux, and on Windows an NSIS installer with `!define WAILS_INSTALL_SCOPE "user"`, installing to `$LOCALAPPDATA\Programs\Mortar` without admin (`project.nsi.tmpl:32,77`). No deb or rpm: the updater cannot replace a root-owned binary. No Flatpak target exists in Wails. Windows builds ship unsigned, so SmartScreen warns and the download page explains it.
- **Linux desktop integration:** nothing installs an AppImage's embedded `.desktop` file, so on first run Mortar writes `~/.local/share/applications/mortar.desktop` with `Exec=<resolved AppImage path> %u` (desktop files do not expand variables) and `MimeType=x-scheme-handler/mortar;application/x-mortar;`, runs `xdg-mime default mortar.desktop x-scheme-handler/mortar`, and rewrites it when the AppImage moves.
- **Self-update:** Wails' updater (`app.Updater`, `pkg/updater`) with the `endpoint` provider reading a signed `manifest.json` published as a GitHub release asset, fetched from the fixed URL `https://github.com/Rethunk-AI/mortar/releases/latest/download/manifest.json` (`wails3 updater genkey`, `sign` and `manifest`, `internal/commands/updater_tool.go:33-43`); the public key ships in the app. The GitHub provider is not used, since it verifies no signature, only an optional checksum asset (`providers/github/github.go:51-55`). Assets are named `mortar-linux-x86_64.AppImage` and `mortar-windows-amd64.exe`. Three traps need patches to the updater, sent upstream as a second PR: it stages in `os.MkdirTemp("")` (`download.go:28`) and then does a plain `os.Rename` (`helper_unix.go:26-30`), which fails across filesystems, and `/tmp` is tmpfs on Fedora, so it must stage beside the target or copy on `EXDEV`; it targets and spawns its helper from `os.Executable()` (`updater.go:407-430`, `updater_notdarwin.go:5`), which inside an AppImage is the read-only mount, so both must use `$APPIMAGE`; and after an update on Windows, `DisplayVersion` under the uninstall key is stale, so Mortar rewrites it.
- The repo turns public at the first release and builds go on its GitHub Releases, since the updater's manifest and assets must be publicly downloadable.

## Tests

Go tests run against local HTTP test servers replaying recorded Nexus, GitHub, SMAPI API and dataset responses, inside the 10 s warm / 30 s cold gate budget. Recordings made with NOMAD's key are scrubbed before commit: `users/validate.json` returns the account's name and email, so fixtures carry placeholder account fields. CI runs on Linux runners only: Wails builds Windows from Linux (`wails3 build GOOS=windows`, `docs/mpress/content/guides/build/building.md:21-29`; only macOS and Linux targets need Docker), and the NSIS installer is built with `makensis`. An opt-in smoke test copies a real Stardew install to scratch space, installs SMAPI with `--no-prompt`, launches a profile, and reads `SMAPI-latest.txt` for the loaded mods, as the planning tests did.

## Build order

After the go-ahead, each milestone ends with the gate green and NOMAD clicking through it on Linux. The screens each milestone builds are the mid-fi mocks (private canvas "Mortar screens (mid-fi)"; `docs/gui-design.md` is the canonical spec):

1. **Shell and look.** Remaining: when wailsapp/wails#6197 (GTK4 transparency) ships in a tagged v3 beta, pin that beta and drop the `Rethunk-AI/wails` replace in `go.mod`.
2. **Stardew core.** SMAPI install and updates (GitHub releases), bundled mods, profiles (duplicate, delete to trash, toggle, copy into `mods/`), installing from a picked or dropped archive (the drop overlay), the Grid and List views, profile management, first run (all steps), launch with success detection and the Launching screen (the Windows path is written here and measured in milestone 6), the Console tab, and the Notes tab.
3. **Mod data.** Manifest scanning, dependency and problem checks (with the duplicate resolver), SMAPI API update checks and Update review, mod detail, update and rollback with carry-over and save backups, the dataset, the save scan and the Saves tab.
4. **Nexus.** Personal-key sign-in (Settings › Nexus Mods) and the account section at the foot of the Mortar drawer (the Nexus account name, Premium or not, Sign in or Sign out; NOMAD, 2026-09-30), runtime `nxm://` registration and its notifications, the guided download queue, GitHub mod sources.
5. **Sharing.** Links, the Share dialog, the import preview, `.mortar` files, and the static page.
6. **Release.** smapi.io/log upload and the GitHub issue link, the updater patches and signed manifest, Windows measurements and fixes, packaging, the repo made public.

## Later

Not in the first release, each by NOMAD on 2026-09-29; re-weigh only when asked:

- Game Select hover in the style of Concrete's switcher, where the hovered game grows while the others become strips, with parallax and brightness shifts (NOMAD, 2026-09-30): eased `flex-grow` and `flex-basis` looked jumpy in WebKitGTK, so it was removed; revisit with a measured, transform-based approach.
- Lethal Company, as the second `Game` implementation ([lethal-company.md](lethal-company.md)).
- Translucent window, desktop showing through (NOMAD, 2026-09-30, deferred to v2). The see-through window looked wrong, so v1 is solid. The `Rethunk-AI/wails` fork's GTK4 `setTransparent()` fix (upstream wailsapp/wails#6197) makes it possible: stock GTK4 leaves `setTransparent()` empty (`v3/pkg/application/linux_cgo.go:1418`), and the fix registers a display-wide CSS provider that clears the window background except the title bar. Any fading or `backdrop-filter` full-window layer turns WebKitGTK's translucent window opaque; a static tint does not. Stacked alphas compound toward opaque, so images and overlays each need their own alpha. Frosted glass needs `ext-background-effect-v1`, below.
- Frosted glass on Linux (NOMAD, 2026-09-30), built when NOMAD's desktop runs GNOME 51. CSS cannot do it: `backdrop-filter` sees only the webview's pixels, and a full-window filter layer turns WebKitGTK's translucent window opaque. The compositor blurs behind the window through the Wayland protocol `ext-background-effect-v1` (Mutter from GNOME 51, KWin from Plasma 6.7; GTK 4.23.3 speaks it). Shape: in the `Rethunk-AI/wails` fork, beside `setTransparent()` in `v3/pkg/application/linux_cgo.go:1418`, bind `ext_background_effect_manager_v1` on Wayland, and when it advertises blur, set the toplevel `wl_surface`'s blur region to the whole window, updated on resize; a no-op elsewhere, and only while the translucent window is on. Accept when the desktop behind the window shows blurred on GNOME 51 and is unchanged on GNOME 50. Offer it upstream with the GTK4 transparency PR. Windows already blurs through Acrylic.
- A hosted share service with short codes and share versioning (running costs).
- Importing the game folder's existing `Mods` folder as a profile.
- In-app mod search and browsing: Mortar links out to Nexus.
- CurseForge (needs an API key application) and ModDrop (no documented download API) as sources; the Xbox app version; GOG and launchers such as Heroic and Lutris; Steam Deck; Flatpak Steam.
- Windows code signing.
- Per-profile save isolation.
- Registering Mortar with Nexus (SSO slug; ask then about OAuth, which Vortex uses via `nxm://oauth/callback`, and Collections), and a mode for users without an API key: an `nxm://` link cannot become a download without API authentication (HTTP 401 without a key, measured), so that mode would pick up manual downloads from the Downloads folder by their manifests, with confirmation.
