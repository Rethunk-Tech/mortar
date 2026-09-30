# Mortar: decided, not yet built

This file holds only work that is decided but not built. Each item is written to be implemented cold: the shape, the evidence it rests on, the traps, and when it is done. When an item lands, delete it here and move any fact that stays true to [architecture.md](architecture.md); the code is the record of what exists. Screens and styling: [gui-design.md](gui-design.md). Lethal Company research for the second game: [lethal-company.md](lethal-company.md). Standing rules: [AGENTS.md](../AGENTS.md).

Items marked **Measure** need a throwaway test first; those tests run outside this repo and only their results land here. Measurements were taken as stated in architecture.md.

## Sharing (milestone 5): the static page deploy

Links, the Share dialog, the import preview, `.mortar` files and first run's link card are built ([architecture.md](architecture.md#sharing)), and so is the page's source (`site/stardew/p/`). Remaining: deploying it at `https://mortar.rethunk.tech/stardew/p`.

- **Deploy:** `site/stardew/p/index.html` in this repo, as a DigitalOcean App Platform static site (free tier: three static apps); `rethunk.tech` is on DigitalOcean's nameservers, and `maitre.rethunk.tech` is already a CNAME to an App Platform app, so the subdomain is set up the same way. The page shows two buttons, since it cannot tell whether a scheme handler exists: open in Mortar (`mortar://stardew/p/<payload>`) and download Mortar, which also copies the link so the importer can take it after installing.
- **Done when** the link opens the page on the live domain, its button opens Mortar's Import with the link filled in, and no request the page makes carries the fragment.

## Release (milestone 6)

- **Support:** the Console tab's **Get help** uploads the current SMAPI log to smapi.io/log, the Stardew community's standard support tool, and copies the link; a second button opens a prefilled GitHub issue for Mortar's own bugs once the repo is public. The upload is `POST https://smapi.io/log` with a form-encoded body `input=<log text>`, no auth; the server answers with a redirect to `/log/<id>`, which is the link, and answers a failed or empty upload with HTTP 200 and its page, so success means a redirect and nothing else (`Pathoschild/SMAPI` `src/SMAPI.Web/Controllers/LogParserController.cs`). A SMAPI log holds local paths, including the user's name, and anyone with the link can read it, so Mortar shows the log and asks first.
- **Packaging:** an AppImage on Linux, and on Windows an NSIS installer built with `INSTALL_SCOPE=user` (`build/windows/Taskfile.yml:83` passes `-DWAILS_INSTALL_SCOPE=user`), installing to `$LOCALAPPDATA\Programs\Mortar` without admin (`build/windows/nsis/project.nsi:71-72`).
- **Not packaged:** no deb or rpm, since the updater cannot replace a root-owned binary; no Flatpak target exists in Wails.
- **Windows** builds ship unsigned, so SmartScreen warns and the download page explains it.
- **Linux desktop integration:** nothing installs an AppImage's embedded `.desktop` file. First run already writes `~/.local/share/applications/mortar.desktop` and the `.mortar` MIME type ([architecture.md](architecture.md#sharing)); remaining is `Exec=<resolved AppImage path> %u` from `$APPIMAGE` instead of `os.Executable()` (desktop files do not expand variables), rewritten when the AppImage moves.
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
4. **Nexus.** Built.
5. **Sharing.** Built except the static page.
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
