# Mortar: decided, not yet built

This file holds only work that is decided but not built. Each item is written to be implemented cold: the shape, the evidence it rests on, the traps, and when it is done. When an item lands, delete it here and move any fact that stays true to [architecture.md](architecture.md); the code is the record of what exists. Screens and styling: [gui-design.md](gui-design.md). Lethal Company research for the second game: [lethal-company.md](lethal-company.md). Standing rules: [AGENTS.md](../AGENTS.md).

Items marked **Measure** need a throwaway test first; those tests run outside this repo and only their results land here. Measurements were taken as stated in architecture.md.

## Sharing (milestone 5): the static page deploy

Links, the Share dialog, the import preview, `.mortar` files and first run's link card are built ([architecture.md](architecture.md#sharing)), and so is the page's source (`site/stardew/p/`) and the download page (`site/download/`). Remaining: deploying both at `https://mortar.rethunk.tech/stardew/p` and `https://mortar.rethunk.tech/download/`, held back on purpose until the first release is ready (NOMAD, 2026-09-30).

- **Deploy:** `site/stardew/p/index.html` and `site/download/index.html` in this repo, as a DigitalOcean App Platform static site (free tier: three static apps); `rethunk.tech` is on DigitalOcean's nameservers, and `maitre.rethunk.tech` is already a CNAME to an App Platform app, so the subdomain is set up the same way. The share page shows two buttons, since it cannot tell whether a scheme handler exists: open in Mortar (`mortar://stardew/p/<payload>`) and Get Mortar (`/download/`), which also copies the link so the importer can take it after installing.
- **Done when** the link opens the page on the live domain, its button opens Mortar's Import with the link filled in, and no request the page makes carries the fragment.

## Release (milestone 6)

Built: packaging, self-update, the signing key and Linux desktop integration ([architecture.md](architecture.md#release)). Remaining:

- The updater fixes are offered upstream as wailsapp/wails#6200 (EXDEV staging), #6201 (AppImage) and #6202 (OnUpdateApplied, draft pending a WEP); Mortar pins the fork until they ship in a tagged v3 beta; then pin that beta and drop the `Rethunk-AI/wails` replace in `go.mod`. wailsapp/wails#6197 (GTK4 transparency) does not gate this: it only serves the translucent window below.
- The repo turns public at the first release and builds go on its GitHub Releases, since the updater's manifest and assets must be publicly downloadable.
- **Measure on Windows:** how launch arguments order around `%command%`, and whether SMAPI needs `--no-terminal`; frameless window under KDE, and Acrylic with a frameless window; the desktop wallpaper read (`SPI_GETDESKWALLPAPER`); one real update through the updater, and `DisplayVersion` after it.

## Gap-filling features (NOMAD, 2026-09-30)

Chosen from suggestion rounds; each lands as its own unit and moves to architecture.md and gui-design.md when built. Two are left:

- **Error badges on mods:** a badge on each mod that logged errors or warnings in the profile's last run. The pieces exist: `launch.Summarize` reads errors per mod from a run's log, every run's log is stored under the profile's `runs/` ([architecture.md](architecture.md#launch)) and the crash dialog lists the mods; what is missing is showing the newest run's per-mod counts in the list, grid and sidebar, and clearing a badge when a later run is clean.
- **Profile change history with revert:** a bounded log per profile of what changed (mods added, removed, updated, rolled back, switched on or off) with a Revert that restores an earlier state. Nothing records changes today; `previousKey` holds only one step of rollback per entry.

## Build order

After the go-ahead, each milestone ends with the gate green and NOMAD clicking through it on Linux. `docs/gui-design.md` is the canonical spec for the screens each milestone builds:

1. **Shell and look.** Built.
2. **Stardew core.** Built.
3. **Mod data.** Built.
4. **Nexus.** Built.
5. **Sharing.** Built except the static page.
6. **Release.** Built except Windows measurements and fixes, and the repo made public.

## Later

Not in the first release, each by NOMAD on 2026-09-29; re-weigh only when asked:

- A hands-on test of updating a mod in place (dropping a newer archive over an older installed version: carry-over, `.mortar-old`, save backup, Roll back) (NOMAD, 2026-09-30, deferred to v2): the engine is built and unit-tested, but no older mod version was at hand to try it with.
- Game Select hover in the style of Concrete's switcher, where the hovered game grows while the others become strips, with parallax and brightness shifts (NOMAD, 2026-09-30): eased `flex-grow` and `flex-basis` looked jumpy in WebKitGTK, so v1 has no hover effect; revisit with a measured, transform-based approach.
- Lethal Company, as the second `Game` implementation ([lethal-company.md](lethal-company.md)).
- Translucent window, desktop showing through (NOMAD, 2026-09-30, deferred to v2). The see-through window looked wrong, so v1 is solid. The `Rethunk-AI/wails` fork's GTK4 `setTransparent()` fix (upstream wailsapp/wails#6197) makes it possible: stock GTK4 leaves `setTransparent()` empty (`wailsapp/wails` `v3/pkg/application/linux_cgo.go:1418`), and the fix registers a display-wide CSS provider that clears the window background except the title bar. Any fading or `backdrop-filter` full-window layer turns WebKitGTK's translucent window opaque; a static tint does not. Stacked alphas compound toward opaque, so images and overlays each need their own alpha. Frosted glass needs `ext-background-effect-v1`, below.
- Frosted glass on Linux (NOMAD, 2026-09-30), built when NOMAD's desktop runs GNOME 51. CSS cannot do it: `backdrop-filter` sees only the webview's pixels, and a full-window filter layer turns WebKitGTK's translucent window opaque. The compositor blurs behind the window through the Wayland protocol `ext-background-effect-v1` (Mutter from GNOME 51, KWin from Plasma 6.7; GTK 4.23.3 speaks it). Shape: in the `Rethunk-AI/wails` fork, beside `setTransparent()` in `v3/pkg/application/linux_cgo.go`, bind `ext_background_effect_manager_v1` on Wayland, and when it advertises blur, set the toplevel `wl_surface`'s blur region to the whole window, updated on resize; a no-op elsewhere, and only while the translucent window is on. Accept when the desktop behind the window shows blurred on GNOME 51 and is unchanged on GNOME 50. Offer it upstream with the GTK4 transparency PR. Windows already blurs through Acrylic.
- A hosted share service with short codes and share versioning (running costs).
- In-app mod search and browsing: Mortar links out to Nexus.
- CurseForge (needs an API key application) and ModDrop (no documented download API) as sources; the Xbox app version; GOG and launchers such as Heroic and Lutris; Steam Deck; Flatpak Steam.
- Windows code signing.
- Per-profile save isolation.
- Registering Mortar with Nexus (SSO slug; ask then about OAuth, which Vortex uses via `nxm://oauth/callback`, and Collections), and a mode for users without an API key: an `nxm://` link cannot become a download without API authentication (HTTP 401 without a key, measured), so that mode would pick up manual downloads from the Downloads folder by their manifests, with confirmation.
