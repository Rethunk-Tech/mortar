# Mortar: decided, not yet built

This file holds only work that is decided but not built. Each item is written to be implemented cold: the shape, the evidence it rests on, the traps, and when it is done. When an item lands, delete it here and move any fact that stays true to [architecture.md](architecture.md); the code is the record of what exists. Screens and styling: [gui-design.md](gui-design.md). Lethal Company research for the second game: [lethal-company.md](lethal-company.md). Standing rules: [AGENTS.md](../AGENTS.md).

Items marked **Measure** need a throwaway test first; those tests run outside this repo and only their results land here. Measurements were taken as stated in architecture.md.

## Nexus download queue (milestone 4)

- **Guided queue for free accounts:** an import or a profile's missing files run a queue. Mortar opens each missing file's download page in turn, `https://www.nexusmods.com/stardewvalley/mods/<id>?tab=files&file_id=<file id>&nmm=1`, the URL Vortex opens for free accounts (`Nexus-Mods/Vortex` `src/renderer/src/extensions/nexus_integration/eventHandlers.ts`), takes the `nxm://` link from the user's click, downloads and installs it, and opens the next. An expired key reopens its page. Premium accounts download without clicks (facts: [architecture.md](architecture.md#nexus-mods)).
- **Failure:** Nexus unreachable or erroring pauses the queue and retries with backoff; a reached rate limit waits for the reset time and shows it.
- **`nxm://` routing:** a valid link (rules in [architecture.md](architecture.md#trust-boundaries)) whose mod and file match a waiting queue item completes that item instead of showing the arrival card.
- Screen: [gui-design.md](gui-design.md#download-queue).

## GitHub sources (milestone 4)

An `UpdateKeys` entry `GitHub:<owner>/<repo>` maps to `/repos/<owner>/<repo>/releases`, read without sign-in (60 calls an hour per IP, so results are cached, update checks go through SMAPI's API first, and a reached limit shows its retry time). An install or update takes the release's only archive asset, or asks when there are several. A GitHub reference in a link is marked unverified in the preview; after download, the mod installs only if SMAPI's update API (`includeExtendedMetadata: true`) gives the manifest's `UniqueID` the same `metadata.gitHubRepo`, otherwise the user sees the mismatch and decides. GitHub being unreachable degrades the feature and never blocks.

## Sharing (milestone 5)

- **Form:** Discord makes only `http://`, `https://` and `discord://` links clickable (Discord community posts 14574789338135 and 25441082215447), so a shared link is `https://mortar.rethunk.tech/stardew/p#<payload>`. The page is a static file, `site/stardew/p/index.html` in this repo, deployed as a DigitalOcean App Platform static site (free tier: three static apps); `rethunk.tech` is on DigitalOcean's nameservers, and `maitre.rethunk.tech` is already a CNAME to an App Platform app, so the subdomain is set up the same way. Browsers never send the part after `#`, so no profile data reaches any server. The page shows two buttons, since it cannot tell whether a scheme handler exists: open in Mortar (`mortar://stardew/p/<payload>`) and download Mortar, which also copies the link so the importer can take it after installing. The app accepts either form pasted into its import box.
- **Payload:** compact JSON `[1, "<profile name>", [<entry>, ...]]`, the leading number being the format version (an older Mortar meeting a newer one says to update), Brotli-compressed at quality 11 and base64url-encoded without padding. An entry is `[<nexus mod id>, <nexus file id>]`, or `"<owner>/<repo>@<tag>/<asset>"` for GitHub, so a link names the exact file. Only enabled entries with a source are shared; entries from local archives are listed at Share as left out. SMAPI's bundled mods (the `smapi-<version>` entry) and the bridge (`bridge-<version>`) are never shared, listed or imported: every profile gets them from the local SMAPI install (NOMAD, 2026-09-30). Measured on real mod IDs sampled from the mod dataset: the whole link is 498 characters for 50 mods, 879 for 100 and 1,630 for 200, so about 240 mods fit a 2,000-character Discord message. `UniqueID`s would push 100 mods to 3,435 characters, since they barely compress, so the link has none. Brotli beats the standard library's deflate by 12 to 15% at every size.
- **Import:** the preview matches entries the user already has by Nexus mod and file ID (from `profile.json`), resolves the rest through the mod dataset to `UniqueID`s and dependencies, groups mods as installed, to download, dependencies added, checked after download (files the dataset lacks), and unavailable (with the page link); importing installs everything available and leaves the unavailable ones listed on the profile. A shared file Nexus has since deleted or archived is replaced by the `MAIN` file with the same version, else the primary file, marked "different file".
- **`.mortar` file:** a zip holding `profile.json` (the same entries as a link) and `configs/<UniqueID>/<relative path>.json` for each enabled mod's `.json` files. Share always offers it as "with settings" and suggests it over about 240 mods. It opens by drop, file picker, or double-click (a `.mortar` file association registered by the installer on Windows and by the `.desktop` file on Linux).
- **Link routing:** `mortar://` goes in `protocols:` in `build/config.yml`, which the NSIS installer registers on Windows. `Options.SingleInstance` with `OnSecondInstanceLaunch`: a link reaches the app as `events.Common.ApplicationLaunchedWithUrl` on a cold start (fired when the only argument contains `://`, `Rethunk-AI/wails` `v3/pkg/application/application_linux.go:73-83`) or in `SecondInstanceData.Args` when it is already running. Both feed one link router.
- **New Go dependency:** `github.com/andybalholm/brotli` (MIT, maintained; the standard library has no Brotli).
- **Trust boundaries:** link parsing accepts only the known forms and caps the encoded payload at 8 KB and the decompressed payload at 64 KB before parsing; nothing downloads or launches from a link by itself, every import shows a preview and waits; a `.mortar` file writes only `.json` files inside the mod folders its profile installs; the clipboard is read only when the user presses Import. Extraction limits: [architecture.md](architecture.md#trust-boundaries).
- Screens: Share and Import in [gui-design.md](gui-design.md#profile-management).

## Release (milestone 6)

- **Support:** the Console tab's **Get help** uploads the current SMAPI log to smapi.io/log, the Stardew community's standard support tool, and copies the link; a second button opens a prefilled GitHub issue for Mortar's own bugs once the repo is public. The upload is `POST https://smapi.io/log` with a form-encoded body `input=<log text>`, no auth; the server answers with a redirect to `/log/<id>`, which is the link, and answers a failed or empty upload with HTTP 200 and its page, so success means a redirect and nothing else (`Pathoschild/SMAPI` `src/SMAPI.Web/Controllers/LogParserController.cs`). A SMAPI log holds local paths, including the user's name, and anyone with the link can read it, so Mortar shows the log and asks first.
- **Packaging:** an AppImage on Linux, and on Windows an NSIS installer built with `INSTALL_SCOPE=user` (`build/windows/Taskfile.yml:83` passes `-DWAILS_INSTALL_SCOPE=user`), installing to `$LOCALAPPDATA\Programs\Mortar` without admin (`build/windows/nsis/project.nsi:71-72`).
- **Not packaged:** no deb or rpm, since the updater cannot replace a root-owned binary; no Flatpak target exists in Wails.
- **Windows** builds ship unsigned, so SmartScreen warns and the download page explains it.
- **Linux desktop integration:** nothing installs an AppImage's embedded `.desktop` file, so on first run Mortar writes `~/.local/share/applications/mortar.desktop` with `Exec=<resolved AppImage path> %u` (desktop files do not expand variables) and `MimeType=x-scheme-handler/mortar;application/x-mortar;`, runs `xdg-mime default mortar.desktop x-scheme-handler/mortar`, and rewrites it when the AppImage moves.
- **Self-update:** Wails' updater (`app.Updater`, `pkg/updater`) with the `endpoint` provider reading a signed `manifest.json` published as a GitHub release asset, fetched from the fixed URL `https://github.com/Rethunk-AI/mortar/releases/latest/download/manifest.json` (`wails3 updater genkey`, `sign` and `manifest`, `Rethunk-AI/wails` `v3/internal/commands/updater_tool.go`). The public key ships in the app. Assets are named `mortar-linux-x86_64.AppImage` and `mortar-windows-amd64.exe`.
- **Why not the GitHub provider:** it verifies no signature, only an optional checksum asset (`v3/pkg/updater/providers/github/github.go:51-55`).
- **Three traps need patches to the updater**, sent upstream as a second PR:
  - It stages in `os.MkdirTemp("")` (`v3/pkg/updater/download.go:28`) and then does a plain `os.Rename` (`helper_unix.go:26-30`), which fails across filesystems, and `/tmp` is tmpfs on Fedora, so it must stage beside the target or copy on `EXDEV`.
  - It targets and spawns its helper from `selfExecutable()` (`updater.go:407`, `spawn.go:12-14`), which is `os.Executable()` and inside an AppImage is the read-only mount, so both must use `$APPIMAGE`.
  - After an update on Windows, `DisplayVersion` under the uninstall key is stale, so Mortar rewrites it.
- The repo turns public at the first release and builds go on its GitHub Releases, since the updater's manifest and assets must be publicly downloadable.
- **CI:** Linux runners only: Wails builds Windows from Linux (`wails3 build GOOS=windows`, `Rethunk-AI/wails` `docs/mpress/content/guides/build/building.md:21-29`; only macOS and Linux targets need Docker), and the NSIS installer is built with `makensis`.
- **Measure on Windows:** how launch arguments order around `%command%`, and whether SMAPI needs `--no-terminal`; frameless window under KDE, and Acrylic with a frameless window; the desktop wallpaper read (`SPI_GETDESKWALLPAPER`).

## Build order

After the go-ahead, each milestone ends with the gate green and NOMAD clicking through it on Linux. `docs/gui-design.md` is the canonical spec for the screens each milestone builds:

1. **Shell and look.** Remaining: when wailsapp/wails#6197 (GTK4 transparency) ships in a tagged v3 beta, pin that beta and drop the `Rethunk-AI/wails` replace in `go.mod`.
2. **Stardew core.** Built.
3. **Mod data.** Built.
4. **Nexus.** Remaining: the guided download queue and its queue icon, GitHub mod sources. Built: personal-key sign-in, the Mortar drawer's account section, runtime `nxm://` registration and its notifications.
5. **Sharing.** Links, the Share dialog, the import preview, `.mortar` files, the static page, and first run's **From a shared link** card.
6. **Release.** smapi.io/log upload and the GitHub issue link, the updater patches and signed manifest, Settings › Updates, Check for updates and Report a bug in the app menu, Windows measurements and fixes, packaging, the repo made public.

## Later

Not in the first release, each by NOMAD on 2026-09-29; re-weigh only when asked:

- A hands-on test of updating a mod in place (dropping a newer archive over an older installed version: carry-over, `.mortar-old`, save backup, Roll back) (NOMAD, 2026-09-30, deferred to v2): the engine is built and unit-tested, but no older mod version was at hand to try it with.
- Game Select hover in the style of Concrete's switcher, where the hovered game grows while the others become strips, with parallax and brightness shifts (NOMAD, 2026-09-30): eased `flex-grow` and `flex-basis` looked jumpy in WebKitGTK, so v1 has no hover effect; revisit with a measured, transform-based approach.
- Lethal Company, as the second `Game` implementation ([lethal-company.md](lethal-company.md)).
- Translucent window, desktop showing through (NOMAD, 2026-09-30, deferred to v2). The see-through window looked wrong, so v1 is solid. The `Rethunk-AI/wails` fork's GTK4 `setTransparent()` fix (upstream wailsapp/wails#6197) makes it possible: stock GTK4 leaves `setTransparent()` empty (`wailsapp/wails` `v3/pkg/application/linux_cgo.go:1418`), and the fix registers a display-wide CSS provider that clears the window background except the title bar. Any fading or `backdrop-filter` full-window layer turns WebKitGTK's translucent window opaque; a static tint does not. Stacked alphas compound toward opaque, so images and overlays each need their own alpha. Frosted glass needs `ext-background-effect-v1`, below.
- Frosted glass on Linux (NOMAD, 2026-09-30), built when NOMAD's desktop runs GNOME 51. CSS cannot do it: `backdrop-filter` sees only the webview's pixels, and a full-window filter layer turns WebKitGTK's translucent window opaque. The compositor blurs behind the window through the Wayland protocol `ext-background-effect-v1` (Mutter from GNOME 51, KWin from Plasma 6.7; GTK 4.23.3 speaks it). Shape: in the `Rethunk-AI/wails` fork, beside `setTransparent()` in `v3/pkg/application/linux_cgo.go`, bind `ext_background_effect_manager_v1` on Wayland, and when it advertises blur, set the toplevel `wl_surface`'s blur region to the whole window, updated on resize; a no-op elsewhere, and only while the translucent window is on. Accept when the desktop behind the window shows blurred on GNOME 51 and is unchanged on GNOME 50. Offer it upstream with the GTK4 transparency PR. Windows already blurs through Acrylic.
- A hosted share service with short codes and share versioning (running costs).
- Importing the game folder's existing `Mods` folder as a profile.
- In-app mod search and browsing: Mortar links out to Nexus.
- CurseForge (needs an API key application) and ModDrop (no documented download API) as sources; the Xbox app version; GOG and launchers such as Heroic and Lutris; Steam Deck; Flatpak Steam.
- Windows code signing.
- Per-profile save isolation.
- Registering Mortar with Nexus (SSO slug; ask then about OAuth, which Vortex uses via `nxm://oauth/callback`, and Collections), and a mode for users without an API key: an `nxm://` link cannot become a download without API authentication (HTTP 401 without a key, measured), so that mode would pick up manual downloads from the Downloads folder by their manifests, with confirmation.
