# Mortar: decided, not yet built

This file holds only work that is decided but not built. Each item is written to be implemented cold: the shape, the sources it rests on, the traps, and when it is done. When an item lands, delete it here; the code is the record of what exists. Standing rules live in [AGENTS.md](../AGENTS.md).

Nothing is built yet. Implementation starts only on NOMAD's explicit go-ahead; until then this file is refined by research and question rounds. Items marked **Open** are still being decided, and items marked **Measure** need a throwaway test before the shape is final.

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

**Open, Linux translucency.** Wails v3's default GTK4 build leaves `setTransparent()` empty (`v3/pkg/application/linux_cgo.go:1418`), so only the webview's background alpha applies and the window itself may paint opaque. The legacy GTK3 build (`-tags gtk3`) sets an RGBA visual when the screen is composited, which is what Concrete got from Wails v2, but GTK3 is removed in v3.1. Blur on Linux comes from the compositor either way (KDE's blur effect, for example), never from the app. Choices: contribute GTK4 transparency upstream to Wails, build with `-tags gtk3` until v3.1, or accept an opaque window on Linux. **Measure:** a throwaway GTK4 window with a transparent CSS background under KDE and GNOME, to see whether upstream work is small.

## Architecture

### Per-game boundary

One Go interface holds everything that differs per game, and nothing else:

- **Discover:** find the install (Steam library folders via `libraryfolders.vdf`, reusing Concrete's `src/app/steam/`; GOG and Xbox paths where the game has them).
- **Loader:** detect, install and update the mod loader.
- **Sources:** resolve a mod reference to a version and a download.
- **Identity:** read a mod's manifest into a shared shape (ID, version, dependencies).
- **Launch:** start the game with a profile.

Stardew Valley is the first implementation and Lethal Company the second. Shared code covers Steam discovery, safe archive extraction (reusing Concrete's `secureArchivePath` and `extractEntry`, with its 256 MiB per-entry cap), the mod store, profiles, share links, the download queue and the UI.

### Storage

All Mortar data lives in the user data directory (`%LOCALAPPDATA%\Mortar`, `$XDG_DATA_HOME/mortar`):

- `store/<game>/<mod-id>/<version>/`: each downloaded mod version, extracted once.
- `profiles/<game>/<profile-id>/`: the profile's own mod tree plus `profile.json`.
- `cache/`: API responses and indexes.

**Open, how a profile holds its mods.** Copying from the store costs disk per profile. Hard links cost nothing extra but need the store and profiles on one volume. Symlinks need Developer Mode on Windows, and whether SMAPI follows them is unmeasured. Trap for any linking scheme: mods write `config.json` into their own folder, so a linked `config.json` would be shared between profiles. It must always be a real per-profile file. Proposed: hard-link every file except `config.json`, and fall back to copying when the volumes differ.

## Stardew Valley

Sources: SMAPI's `docs/technical/smapi.md`, `docs/technical/web.md`, `src/SMAPI.Toolkit/Serialization/Models/Manifest.cs` and `ModScanner.cs`; the Stardew Valley wiki's Modding pages.

- **Discovery:** Steam app `413150`. The wiki lists default game folders for Steam, GOG and the Xbox app on Windows, and Steam and GOG on Linux.
- **SMAPI install and update:** download the release installer (latest 4.5.2, needs Stardew 1.6.14 or later) and run it unattended with `--install --no-prompt --game-path "<dir>"`. Running it again updates SMAPI. On Linux the installer renames the game's `StardewValley` to `StardewValley-original` and puts its launcher in its place, so a game update breaks SMAPI. Mortar detects that at startup and offers the reinstall.
- **Profiles:** SMAPI takes `--mods-path <path>` on Windows and the `SMAPI_MODS_PATH` environment variable on Linux, where command-line arguments do not reach SMAPI (`smapi.md:40-59`). Paths may be absolute.
- **Bundled mods:** a profile folder replaces `Mods/`, so SMAPI's bundled Console Commands and Save Backup mods must be placed into every profile.
- **Scanning:** SMAPI recurses into subfolders until it finds a `manifest.json` and skips any folder whose name starts with a dot. Manifests are JSON with comments and trailing commas, so they are parsed leniently.
- **Dependencies:** `Dependencies[]` (`UniqueID`, `MinimumVersion`, `IsRequired` defaulting to true) plus `ContentPackFor`, treated as a required dependency on the framework mod. Every check is by `UniqueID`.
- **Update checks:** `POST https://smapi.io/api/v4.0.0/mods` with each mod's ID, `UpdateKeys` and installed version returns `suggestedUpdate` and mod page URLs. Take `suggestedUpdate` as given, since it applies SMAPI's release-channel logic. The API is public but officially unreleased and may change, so results are cached and a failure is never fatal.
- **Saves** are shared across profiles in `%APPDATA%\StardewValley\Saves` and `~/.config/StardewValley/Saves`. Mortar does not isolate them in v1.

**Measure, launch through Steam.** Launching `StardewModdingAPI` directly loses Steam's overlay, achievements and playtime tracking. Steam also ignores environment variables set on the `steam -applaunch` process. The proposed shape: Steam launch options are set once to point at a fixed path (`SMAPI_MODS_PATH="<data>/active/stardew" %command%` on Linux; `"<game>\StardewModdingAPI.exe" --mods-path "<data>\active\stardew" %command%` on Windows), and Mortar repoints that path (a symlink, or a junction on Windows) at the chosen profile before each launch. Mortar writes the launch options into `localconfig.vdf` only while Steam is closed, or asks the user to paste them. None of this is verified against SMAPI and Steam yet.

## Nexus Mods

Sources: the Nexus API acceptable-use policy (help.nexusmods.com article 114), the rate-limit article (105), the SSO demo repo `Nexus-Mods/sso-integration-demo`, and the `Nexus-Mods/node-nexus-api` client.

- **Registration:** a public app must email <support@nexusmods.com> with a testing build, and Nexus issues an application slug for SSO. Until then only a personal API key may be used, for testing. So a working build has to exist before public release, and Nexus may refuse.
- **Sign-in:** SSO over `wss://sso.nexusmods.com` with the slug; the user approves in the browser and the socket returns the user's API key, which Mortar keeps in the OS keyring. Keys are never sent to any server of ours, and no call is made that the user did not start.
- **Headers:** every request carries a truthful `Application-Name` and `Application-Version`.
- **Rate limits:** 20,000 requests a day, then 500 an hour, reported in the `x-rl-*` headers. The client backs off as the remaining count gets low.
- **Downloads:** `GET /v1/games/{game}/mods/{mod}/files/{file}/download_link.json`. Premium users get the link directly. Free users need the `key` and `expires` from an `nxm://` link generated by the site's download button, so each mod needs one click on its Nexus page. For free users, a profile import opens each missing mod's files page in turn and completes each download when its `nxm://` link arrives.
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

What a shared profile holds: name, game, and per mod the ID, the version, and the source reference (`Nexus:<mod>` plus file ID, `GitHub:<owner>/<repo>`, or `Namespace-Name-Version`). Optionally, mod configs.

**Open, where the payload lives.** A 100-mod Stardew profile is about 6 KB of JSON, and roughly 3 KB after compression and base64. Discord messages allow 2,000 characters, or 4,000 with Nitro. So a link that carries the whole profile only works for small profiles. Choices:

- Inline for small profiles, and a file (`.mortar`) for larger ones.
- A small hosted store that takes the payload and returns a short code. That means a service to run (fleet default: DigitalOcean App Platform) and an abuse policy.
- A GitHub Gist, which needs the sharer's GitHub sign-in.

For Lethal Company, r2modman codes already cover this through Thunderstore.

## Open questions for NOMAD

- Linux translucency: upstream GTK4 fix, GTK3 tag, or opaque.
- Share payload: inline plus file, hosted short codes, or Gist.
- Stardew mod sources in v1 beyond Nexus: GitHub releases (no auth), ModDrop and CurseForge (CurseForge needs an API key application).
- Free Nexus users: is a guided one-click-per-mod queue acceptable, or is Premium-only automatic install the product?
- What happens to LethalModding/Concrete once Mortar ships Lethal Company support.
