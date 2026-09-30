# Mortar: decided, not yet built

This file holds only work that is decided but not built. Each item is written to be implemented cold: the shape, the sources it rests on, the traps, and when it is done. When an item lands, delete it here; the code is the record of what exists. Standing rules live in [AGENTS.md](../AGENTS.md).

Nothing is built yet. Implementation starts only on NOMAD's explicit go-ahead; until then this file is refined by research and question rounds. Items marked **Open** are still being decided, and items marked **Measure** need a throwaway test before the shape is final. Those tests run outside this repo and only their results land here (NOMAD, 2026-09-29).

## Product

Mortar finds a game, installs its mod loader, keeps each set of mods in its own profile, and launches the game with the chosen profile. The feature that sets it apart is sharing: a profile goes out as a link, and opening it on another machine produces the same mods, installing or queueing whatever is missing and checking every dependency first.

Platforms are Windows and Linux. macOS is out: fleet CI has no macOS runners, and Wails does not cross-compile to it.

## Stack

- **Wails v3**, pinned to `v3.0.0-beta.26` (Go 1.25+, `v3/go.mod:3`) and upgraded one beta at a time on purpose. Recent betas are small fixes (`docs/mpress/content/changelog.md`); there is no date for a stable v3.0.
- **Frontend:** React, TypeScript and MUI on Vite, from `wails3 init`'s React template. Typed bindings come from `wails3 generate bindings` into `frontend/bindings`. Client state in zustand, as in Concrete.
- **Go services:** each backend area is a Wails service (`application.NewService`, optional `ServiceStartup`/`ServiceShutdown`), so its exported methods are bound to TypeScript.
- **Single instance:** `Options.SingleInstance` with `OnSecondInstanceLaunch`, which receives the second launch's `Args`. Links reach the app two ways: `events.Common.ApplicationLaunchedWithUrl` on a cold start, and `SecondInstanceData.Args` when the app is already running. Both paths feed one link router.
- **Link schemes:** `protocols:` in `build/config.yml` registers `mortar://` and `nxm://`. The NSIS installer and MSIX manifest register them on Windows. On Linux, `desktop.tmpl` emits `MimeType=x-scheme-handler/...`. Trap: the default Linux Taskfile's `generate:dotdesktop` calls `wails3 generate .desktop` without `-mimetype`, so the AppImage would not register the schemes until that task is changed.
- **Self-update:** the built-in updater (`app.Updater`, `pkg/updater`) with the GitHub releases provider. It requires a signing key pair; the public key ships in the app.
- **Packaging:** NSIS on Windows; AppImage, deb and rpm on Linux. There is no Flatpak target in Wails.

## Look

Concrete's look carries over unchanged apart from colour tokens:

- The window is translucent: Acrylic backdrop on Windows (`BackgroundTypeTranslucent` plus `Windows.BackdropType: Acrylic`; requires a Windows build that supports backdrops), and a base colour of `rgba(25,25,30,0.8)`.
- The MUI dark theme from Concrete's `frontend/src/styles/darkThemeOptions.ts`: background and paper at 80% opacity, Open Sans (loaded from `@fontsource/open-sans`, not `next/font`), dialog backdrops at 75% black, and the blurred, glowing hero art from `ProfileDetailsPane.tsx`.
- Native scrollbars themed to match.

**Linux translucency (NOMAD, 2026-09-29): fix it upstream.** Wails v3's default GTK4 build leaves `setTransparent()` empty (`v3/pkg/application/linux_cgo.go:1418`), so the window paints opaque. Measured on GNOME Wayland (2026-09-29, beta.26, NOMAD judging by eye at 35% and 80% alpha): the stock GTK4 build is solid; the GTK3 build (`-tags gtk3`) and a patched GTK4 build are both see-through. The patch is about 15 lines in `setTransparent()`, following the `setFrameless` CSS-provider pattern beside it (`linux_cgo.go:1385-1412`): register once, through a `sync.Once`, a display-wide provider with `window.wails-transparent, window.wails-transparent > *:not(.titlebar):not(headerbar) { background: transparent; }`, then add the `wails-transparent` class to the window. Trap: a bare `window > *` selector also makes the system title bar transparent. Mortar sends this to Wails as a PR (read its CONTRIBUTING and AGENTS files first) and builds with `-tags gtk3` until it merges; GTK3 is removed in Wails v3.1. Blur comes only from a compositor that offers it, such as KDE; GNOME has none, so there the window shows the desktop through at 80% opacity without blur.

**Frameless with a themed title bar (NOMAD, 2026-09-29),** as Concrete had: `Frameless: true` removes the system title bar, and the app draws its own, with drag, minimise, maximise and close. Measured on the patched GTK4 build: translucent, no system title bar, and a strip marked `--wails-draggable: drag` moves the window. Trap: dragging works only once the Wails runtime is loaded (`@wailsio/runtime` in the frontend, or `/wails/runtime.js`); without it the drag region does nothing. **Measure:** resize edges and window shadow on a frameless window under GNOME and KDE, and Acrylic with a frameless window on Windows.

## Architecture

### Per-game boundary

One Go interface holds everything that differs per game, and nothing else:

- **Discover:** find the install (Steam library folders via `libraryfolders.vdf`, reusing Concrete's `src/app/steam/`; GOG and Xbox paths where the game has them).
- **Loader:** detect, install and update the mod loader.
- **Sources:** resolve a mod reference to a version and a download.
- **Identity:** read a mod's manifest into a shared shape (ID, version, dependencies).
- **Launch:** start the game with a profile.

Stardew Valley is the first implementation and ships alone in the first release; Lethal Company is the second and follows in a later version (NOMAD, 2026-09-29). Shared code covers Steam discovery, safe archive extraction (reusing Concrete's `secureArchivePath` and `extractEntry`, with its 256 MiB per-entry cap), the mod store, profiles, share links, the download queue and the UI.

### Storage

All Mortar data lives in the user data directory (`%LOCALAPPDATA%\Mortar`, `$XDG_DATA_HOME/mortar`):

- `store/<game>/<mod-id>/<version>/`: each downloaded mod version, extracted once.
- `profiles/<game>/<profile-id>/`: the profile's own mod tree plus `profile.json`.
- `cache/`: API responses and indexes.

**How a profile holds its mods.** Copying from the store costs disk per profile. Measured on Linux (2026-09-29, SMAPI 4.5.2, Stardew 1.6.15): SMAPI loads a mod whose folder is a symlink into the store and a mod whose files are hard links, and skips a dot-prefixed folder. Trap for any linking scheme: mods write `config.json` into their own folder, so a linked `config.json` would be shared between profiles, and a symlinked folder would carry every profile's config. So each mod folder in a profile is a real directory whose files are hard links to the store, except `config.json`, which is always a real per-profile file. Copy when the store and the profile are on different volumes. **Measure on Windows:** hard links need NTFS and one volume; symlinks need Developer Mode and are not used.

## Stardew Valley

Sources: SMAPI's `docs/technical/smapi.md`, `docs/technical/web.md`, `src/SMAPI.Toolkit/Serialization/Models/Manifest.cs` and `ModScanner.cs`; the Stardew Valley wiki's Modding pages.

- **Discovery:** Steam app `413150`. The wiki lists default game folders for Steam, GOG and the Xbox app on Windows, and Steam and GOG on Linux.
- **SMAPI install and update:** download the release installer (latest 4.5.2, needs Stardew 1.6.14 or later) and run it unattended with `--install --no-prompt --game-path "<dir>"` (measured on Linux: exit 0, no prompts, launcher replaced, bundled mods added). Running it again updates SMAPI. On Linux the installer renames the game's `StardewValley` to `StardewValley-original` and puts its launcher in its place, so a game update breaks SMAPI. Mortar detects that at startup and offers the reinstall.
- **Profiles:** SMAPI takes `--mods-path <path>`, or the `SMAPI_MODS_PATH` environment variable (`smapi.md:40-59`); on Linux the argument must follow `--` (see Launch below). Paths may be absolute. Measured on Linux: `SMAPI_MODS_PATH=<absolute path>` set on the game's `StardewValley` launcher loads mods from that folder.
- **Bundled mods:** a profile folder replaces `Mods/`, so SMAPI's bundled Console Commands and Save Backup mods must be placed into every profile.
- **Scanning:** SMAPI recurses into subfolders until it finds a `manifest.json` and skips any folder whose name starts with a dot. Manifests are JSON with comments and trailing commas, so they are parsed leniently.
- **Dependencies:** `Dependencies[]` (`UniqueID`, `MinimumVersion`, `IsRequired` defaulting to true) plus `ContentPackFor`, treated as a required dependency on the framework mod. Every check is by `UniqueID`.
- **Update checks:** `POST https://smapi.io/api/v4.0.0/mods` with each mod's ID, `UpdateKeys` and installed version returns `suggestedUpdate` and mod page URLs. Take `suggestedUpdate` as given, since it applies SMAPI's release-channel logic. The API is public but officially unreleased and may change, so results are cached and a failure is never fatal.
- **Saves (NOMAD, 2026-09-29): shared, with a warning.** Saves live in one folder, `%APPDATA%\StardewValley\Saves` or `~/.config/StardewValley/Saves`, which Steam Cloud syncs, so Mortar never moves or swaps it. Mortar records which profile last launched each save (by folder modification time after the game exits) and warns before launching a different profile against a save whose last profile had other mods.

**Launch through Steam, as Concrete did (NOMAD, 2026-09-29).** Concrete ran `steam -applaunch <appid> <args>`, and Steam passes the arguments after the app ID to the game, which keeps the overlay, achievements and playtime tracking. Launching `StardewModdingAPI` directly loses those. On Linux, SMAPI's launcher script (`StardewValley` in the game folder) consumes every argument in its own option loop and forwards only what follows `--`, which is why SMAPI's docs say arguments do not work there. Measured (2026-09-29, SMAPI 4.5.2): `./StardewValley --mods-path <profile>` ignores the path, and `./StardewValley -- --mods-path <profile>` loads mods from the profile. So Linux launches with `steam -applaunch 413150 -- --mods-path <absolute profile path>`. On Windows, Steam starts `Stardew Valley.exe`, not SMAPI, so SMAPI's standard setup puts `"<game>\StardewModdingAPI.exe" %command%` in Stardew's Steam launch options once, and Mortar then adds `--mods-path` after the app ID. Measured through NOMAD's real Steam client on Linux (2026-09-29): `steam -applaunch 413150 -- --mods-path <profile>` started SMAPI with `--mods-path`, which loaded the profile's mods, and the game reached the main menu. **Measure on Windows:** how the arguments order around `%command%`.

**SMAPI console inside Mortar.** Without a flag, SMAPI's Linux launcher opens its console in the first terminal it finds, and on NOMAD's GNOME desktop that was `xterm`, too small to read. Mortar passes `--skip-terminal` before the `--` (the launcher then starts SMAPI with `--no-terminal`) and shows `SMAPI-latest.txt` (`~/.config/StardewValley/ErrorLogs/`, `%APPDATA%\StardewValley\ErrorLogs\`) live in a themed console panel, with errors and warnings highlighted. Trap: launching through Steam makes Steam Cloud download the user's saves, and SMAPI's Save Backup mod then zips them into `save-backups/` in the game folder.

## GitHub releases

Stardew v1 installs from Nexus and GitHub (NOMAD, 2026-09-29). An `UpdateKeys` entry `GitHub:<owner>/<repo>` maps to that repo's releases through the public REST API (`/repos/<owner>/<repo>/releases`), without sign-in. Anonymous calls are limited to 60 an hour per IP, so results are cached and update checks go through the SMAPI API first. CurseForge (needs an approved API key) and ModDrop (no documented download API) are later; until then their mods install from a zip the user downloads.

## Nexus Mods

Sources: the Nexus API acceptable-use policy (help.nexusmods.com article 114), the rate-limit article (105), the SSO demo repo `Nexus-Mods/sso-integration-demo`, and the `Nexus-Mods/node-nexus-api` client.

- **Registration:** a public app must email <support@nexusmods.com> with a testing build, and Nexus issues an application slug for SSO. Until then only a personal API key may be used, for testing. So a working build has to exist before public release, and Nexus may refuse.
- **Sign-in:** SSO over `wss://sso.nexusmods.com` with the slug; the user approves in the browser and the socket returns the user's API key, which Mortar keeps in the OS keyring. Keys are never sent to any server of ours, and no call is made that the user did not start.
- **Headers:** every request carries a truthful `Application-Name` and `Application-Version`.
- **Rate limits:** 20,000 requests a day, then 500 an hour, reported in the `x-rl-*` headers. The client backs off as the remaining count gets low.
- **Downloads:** `GET /v1/games/{game}/mods/{mod}/files/{file}/download_link.json`. Premium users get the link directly. Free users need the `key` and `expires` from an `nxm://` link generated by the site's download button, so each mod needs one click on its Nexus page. For free users, a profile import runs a guided queue (NOMAD, 2026-09-29): Mortar opens each missing mod's files page in turn, catches the `nxm://` link from the user's click, installs it, and moves to the next. Premium users skip the queue.
- **nxm:// ownership:** only one app can own the scheme, so registering Mortar takes it from Vortex or Mod Organizer 2. Mortar asks before claiming it.
- **Mapping:** an `UpdateKeys` entry `Nexus:1915` is mod 1915 on the `stardewvalley` domain. The file ID comes from `GET /v1/games/stardewvalley/mods/1915/files.json`, matched by version.
- **No re-hosting:** Mortar never stores or serves mod files or bulk Nexus metadata anywhere but the user's own machine. A shared profile holds IDs only.

**Measure:** the exact `nxm://` link format, read from Vortex's parser before relying on it. **Ask Nexus**, when registering: whether a multi-game third-party manager qualifies, whether OAuth is available instead of the SSO socket, and whether consuming Collections is allowed.

## Lethal Company

Sources: Gale (`Kesomannen/gale`), r2modmanPlus (`ebkr/r2modmanPlus`), the Thunderstore server source (`thunderstore-io/Thunderstore`), and BepInEx 5.4.23.5's release zip.

- **Discovery:** Steam app `1966720`. On Linux it runs through Proton.
- **Mod index:** the chunked listing index at `https://thunderstore.io/c/lethal-company/api/v1/package-listing-index/` (25 gzipped chunks, 50,978 packages, 34.6 MB gzipped, measured 2026-09-30), fetched in parallel and cached. Never the full v1 package list, which is about 332 MB uncompressed.
- **Downloads:** `https://thunderstore.io/package/download/{namespace}/{name}/{version}/`, no auth. Dependencies are `Namespace-Name-Version` strings.
- **Loader:** the `BepInEx-BepInExPack` package. Doorstop's `winhttp.dll` and `doorstop_config.ini` must sit next to the game executable, so, as Gale does (`src-tauri/src/profile/launch/mod.rs:184-225`), launch copies the profile's top-level loader files and `doorstop_libs` into the game folder. The rest of the profile stays in Mortar's data directory.
- **Launch arguments** depend on the Doorstop version, read from the profile's `.doorstop_version` (default 3): `--doorstop-enable true --doorstop-target <preloader>` for 3.x, `--doorstop-enabled true --doorstop-target-assembly <preloader>` for 4.x.
- **Proton:** the Wine override `winhttp=n,b` is written into `compatdata/1966720/pfx/user.reg`, backed up first, as Gale and r2modman both do. Paths passed to Doorstop use the `Z:` form.
- **Package layout:** follows r2modman's rules. `plugins/`, `patchers/`, `monomod/`, `core/` and `config/` (with or without a `BepInEx/` prefix) map to `BepInEx/<dir>/<Namespace-Name>/`. `config/` keeps no per-mod subfolder. Loose DLLs go to `BepInEx/plugins/<Namespace-Name>/`.
- **r2modman profile codes:** import and export them, since that is how Lethal Company players already share. A code is `#r2modman\n` followed by base64 of an `.r2z` zip, which holds `export.r2x` (YAML: `profileName`, `mods[]` with `name`, `version {major,minor,patch}` and `enabled`) and the `config/` folder. Codes are stored with `POST https://thunderstore.io/api/experimental/legacyprofile/create/` and read with `GET .../legacyprofile/get/{key}/`. The endpoint is marked experimental; Gale uses it too.

**Measure:** on Windows, how BepInEx finds its root when the profile lives outside the game folder (r2modman passes `--r2profile` only on Linux).

## Share links

A link names its game: `mortar://<game>/...`. It must open the importer whether Mortar is running or not.

What a shared profile holds: name, game, and per mod its source reference (Nexus mod and file ID, `<owner>/<repo>@<tag>` for GitHub, or `Namespace-Name-Version` for Thunderstore). Mod configs only in the `.mortar` file.

**Self-contained Brotli links; no hosted service in v1 (NOMAD, 2026-09-29).** A hosted share service, and share versioning with it, are parked for v2 to avoid running costs. The link carries the whole profile: `mortar://stardew/p/<payload>`, where the payload is compact JSON (`["<profile name>", [[<nexus mod id>, <nexus file id>], ...]]`), Brotli-compressed at quality 11 and base64url-encoded without padding. Measured on real mod IDs sampled from SMAPI's `StardewModDataset` (`dataset/indexes/pages by mod ID.json`, 2026-09-29), the whole link is 498 characters for 50 mods, 879 for 100 and 1,630 for 200, so profiles up to about 240 mods fit a 2,000-character Discord message. Including SMAPI `UniqueID`s pushes 100 mods to 3,435 characters, because those IDs barely compress, so the link carries none: the recipient reads each `UniqueID` from the downloaded mod's manifest and matches mods it already has through their `Nexus:` update keys. Brotli beats gzip by 12 to 15% at every size. A GitHub-hosted mod adds `"<owner>/<repo>@<tag>"` in place of the ID pair. Mod configs do not fit in a link and travel in an optional `.mortar` file (the same JSON plus configs, uncompressed zip).

For Lethal Company, r2modman codes through Thunderstore stay supported for import and export alongside Mortar codes.


