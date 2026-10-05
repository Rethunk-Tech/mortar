# Mortar: decided, not yet built

This file holds only work that is decided but not built. Each item is written to be implemented cold: the shape, the evidence it rests on, the traps, and when it is done. When an item lands, delete it here and move any fact that stays true to [architecture.md](architecture.md); the code is the record of what exists. Screens and styling: [gui-design.md](gui-design.md). Lethal Company research for the second game: [lethal-company.md](lethal-company.md). Standing rules: [AGENTS.md](../AGENTS.md).

Items marked **Measure** need a throwaway test first; those tests run outside this repo and only their results land here.

## Release

Remaining ([architecture.md](architecture.md#release)):

- The updater fixes are offered upstream as wailsapp/wails#6200 (EXDEV staging), #6201 (AppImage) and #6202 (OnUpdateApplied, draft pending a WEP); Mortar pins the fork until they ship in a tagged v3 beta; then pin that beta and drop the `Rethunk-AI/wails` replace in `go.mod`. wailsapp/wails#6197 (GTK4 transparency) does not gate this: it only serves the translucent window below.
- The repo turns public at the first release and builds go on its GitHub Releases, since the updater's manifest and assets must be publicly downloadable.
- **Measure on Windows:** how launch arguments order around `%command%`, and whether SMAPI needs `--no-terminal`; one real update through the updater, and `DisplayVersion` after it.

## Queued for v1

- **Packages:** an aarch64 Flatpak bundle is blocked without qemu binfmt (or an aarch64 host): `flatpak-builder --arch=aarch64` still runs `build-commands` via the aarch64 SDK's `/bin/sh` (`bwrap: execvp /bin/sh: Exec format error`), even with only `install` of a prebuilt binary.
- **CurseForge** as a third source, after the repository is public: apply for a 3rd-party API key, then build it without caching API data, with a User-Agent on every request, and honouring each author's distribution setting.

## Later

- **UI translations** beyond English, as Stardrop (17+), MO2 and r2modman ship: every string already goes through Lingui and the catalogs are extracted; needs chosen languages and translators. Parked 2026-10-02 (not v1).
- **Steam Deck / gamepad mode** (larger targets, gamepad focus navigation, Game Mode play: a `--play` launch from a Steam shortcut starts Mortar minimised or headless, shows only a small controller-friendly prompt when Play is blocked, and exits when the game closes so Steam tracks playtime): parked 2026-10-02 (not v1).
- **macOS build**: Stardew runs on macOS, and Stardrop ships for x64 and arm64, but Mortar has no macOS CI or test machine; it needs an Apple developer account for signing and notarization, Mac Steam paths and nxm registration, and a Mac to test on. Parked 2026-10-02 (not v1).
- **Accessibility pass** (keyboard-only navigation of every screen, focus order, screen-reader labels on icon buttons, reduced motion everywhere): parked 2026-10-03.
- **One top toolbar** (Gale-style: game switcher and profile switcher with mod count side by side in the title bar, download status next to them): parked 2026-10-05. Play stays at the foot of the sidebar; a Play control in the top-left corner is rejected.
- **Offline mode banner** (clear banner when Nexus/GitHub are unreachable, cached data with "as of" times, network actions disabled with a reason): parked 2026-10-03.

Not in the first release; re-weigh only when asked:

- Game Select hover in the style of Concrete's switcher, where the hovered game grows while the others become strips, with parallax and brightness shifts: eased `flex-grow` and `flex-basis` looked jumpy in WebKitGTK, so v1 has no hover effect; revisit with a measured, transform-based approach.
- Lethal Company, as the second `Game` implementation ([lethal-company.md](lethal-company.md)).
- Translucent window, desktop showing through; on Windows it needs Acrylic measured with the frameless window. The see-through window looked wrong, so v1 is solid. The `Rethunk-AI/wails` fork's GTK4 `setTransparent()` fix (upstream wailsapp/wails#6197) makes it possible: stock GTK4 leaves `setTransparent()` empty (`wailsapp/wails` `v3/pkg/application/linux_cgo.go:1418`), and the fix registers a display-wide CSS provider that clears the window background. Any fading or `backdrop-filter` full-window layer turns WebKitGTK's translucent window opaque; a static tint does not. Stacked alphas compound toward opaque, so images and overlays each need their own alpha. Frosted glass needs `ext-background-effect-v1`, below.
- Frosted glass on Linux, built when the desktop runs GNOME 51. CSS cannot do it: `backdrop-filter` sees only the webview's pixels, and a full-window filter layer turns WebKitGTK's translucent window opaque. The compositor blurs behind the window through the Wayland protocol `ext-background-effect-v1` (Mutter from GNOME 51, KWin from Plasma 6.7; GTK 4.23.3 speaks it). Shape: in the `Rethunk-AI/wails` fork, beside `setTransparent()` in `v3/pkg/application/linux_cgo.go`, bind `ext_background_effect_manager_v1` on Wayland, and when it advertises blur, set the toplevel `wl_surface`'s blur region to the whole window, updated on resize; a no-op elsewhere, and only while the translucent window is on. Accept when the desktop behind the window shows blurred on GNOME 51 and is unchanged on GNOME 50. Offer it upstream with the GTK4 transparency PR. Windows already blurs through Acrylic.
- A hosted share service with short codes and share versioning (running costs).
- ModDrop as a source (no documented download API); the Xbox app version (WindowsApps folders are locked down).
- Windows code signing.
- Bottles as a launcher (Linux): games there are Windows builds in a Wine prefix, so it needs SMAPI's Windows installer run inside the bottle (`bottles-cli run -b <bottle> -e <installer>`) and launches through `bottles-cli run` with `--mods-path`; the Linux SMAPI install would break such a copy. Detection is simple: bottles under `~/.local/share/bottles/bottles` and `~/.var/app/com.usebottles.bottles/data/bottles/bottles`, each searched for `drive_c/Program Files (x86)/Steam` and GOG folders.
- A profile sync folder (Syncthing, Dropbox, a NAS) holding each profile's `.mortar` state, so another machine is offered the changes, with conflict detection when both sides edited; mod files still come from their sources.
- Needs Nexus's approval through app registration first (below), since it starts downloads outside Nexus's own Mod Manager Download button: an "Add to Mortar" button on Nexus listing tiles.
- Per-profile save isolation.
- Settings considered and not taken (2026-10-02): new profiles starting as a copy of the open profile or from a bundle; an offline mode that never contacts the network.
- Registering Mortar with Nexus (Collections still to ask about): SSO and OAuth PKCE code is ready behind a build flag (`nexussso.Slug` / `ClientID`, off while empty), waiting for Nexus approval and a slug or client id; see docs/nexus-application.md.
