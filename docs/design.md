# Mortar: decided, not yet built

This file holds only work that is decided but not built. Each item is written to be implemented cold: the shape, the evidence it rests on, the traps, and when it is done. When an item lands, delete it here and move any fact that stays true to [architecture.md](architecture.md); the code is the record of what exists. Screens and styling: [gui-design.md](gui-design.md). Lethal Company research for the second game: [lethal-company.md](lethal-company.md). Standing rules: [AGENTS.md](../AGENTS.md).

Items marked **Measure** need a throwaway test first; those tests run outside this repo and only their results land here.

## Sharing: the static page deploy

Remaining: deploying `site/stardew/p/` and `site/download/` at `https://mortar.rethunk.tech/stardew/p` and `https://mortar.rethunk.tech/download/`, held back until the first release is ready. How sharing works: [architecture.md](architecture.md#sharing).

- **Deploy:** `site/stardew/p/index.html` and `site/download/index.html` in this repo, as a DigitalOcean App Platform static site (free tier: three static apps); `rethunk.tech` is on DigitalOcean's nameservers, and `maitre.rethunk.tech` is already a CNAME to an App Platform app, so the subdomain is set up the same way. The share page shows two buttons, since it cannot tell whether a scheme handler exists: open in Mortar (`mortar://stardew/p/<payload>`) and Get Mortar (`/download/`), which also copies the link so the importer can take it after installing.
- **Done when** the link opens the page on the live domain, its button opens Mortar's Import with the link filled in, and no request the page makes carries the fragment.

## Release

Remaining ([architecture.md](architecture.md#release)):

- The updater fixes are offered upstream as wailsapp/wails#6200 (EXDEV staging), #6201 (AppImage) and #6202 (OnUpdateApplied, draft pending a WEP); Mortar pins the fork until they ship in a tagged v3 beta; then pin that beta and drop the `Rethunk-AI/wails` replace in `go.mod`. wailsapp/wails#6197 (GTK4 transparency) does not gate this: it only serves the translucent window below.
- The repo turns public at the first release and builds go on its GitHub Releases, since the updater's manifest and assets must be publicly downloadable.
- **Measure on Windows:** how launch arguments order around `%command%`, and whether SMAPI needs `--no-terminal`; one real update through the updater, and `DisplayVersion` after it.

## Queued for v1

- **Library:** an extra folder to scan for mods; a toggle to show dot-hidden mods; asking before deleting old files on update; new folders in the game's own `Mods` folder offered for moving into a profile.
- **Packages:** an aarch64 Flatpak bundle is blocked without qemu binfmt (or an aarch64 host): `flatpak-builder --arch=aarch64` still runs `build-commands` via the aarch64 SDK's `/bin/sh` (`bwrap: execvp /bin/sh: Exec format error`), even with only `install` of a prebuilt binary.
- **Self-test:** update a mod in place with an older GitHub release in a copied game folder, checking carry-over, `.mortar-old`, the save backup and Roll back.
- **CurseForge** as a third source, after the repository is public: apply for a 3rd-party API key, then build it without caching API data, with a User-Agent on every request, and honouring each author's distribution setting.

## Later

Not in the first release; re-weigh only when asked:

- Game Select hover in the style of Concrete's switcher, where the hovered game grows while the others become strips, with parallax and brightness shifts: eased `flex-grow` and `flex-basis` looked jumpy in WebKitGTK, so v1 has no hover effect; revisit with a measured, transform-based approach.
- Lethal Company, as the second `Game` implementation ([lethal-company.md](lethal-company.md)).
- Translucent window, desktop showing through; on Windows it needs Acrylic measured with the frameless window. The see-through window looked wrong, so v1 is solid. The `Rethunk-AI/wails` fork's GTK4 `setTransparent()` fix (upstream wailsapp/wails#6197) makes it possible: stock GTK4 leaves `setTransparent()` empty (`wailsapp/wails` `v3/pkg/application/linux_cgo.go:1418`), and the fix registers a display-wide CSS provider that clears the window background except the title bar. Any fading or `backdrop-filter` full-window layer turns WebKitGTK's translucent window opaque; a static tint does not. Stacked alphas compound toward opaque, so images and overlays each need their own alpha. Frosted glass needs `ext-background-effect-v1`, below.
- Frosted glass on Linux, built when the desktop runs GNOME 51. CSS cannot do it: `backdrop-filter` sees only the webview's pixels, and a full-window filter layer turns WebKitGTK's translucent window opaque. The compositor blurs behind the window through the Wayland protocol `ext-background-effect-v1` (Mutter from GNOME 51, KWin from Plasma 6.7; GTK 4.23.3 speaks it). Shape: in the `Rethunk-AI/wails` fork, beside `setTransparent()` in `v3/pkg/application/linux_cgo.go`, bind `ext_background_effect_manager_v1` on Wayland, and when it advertises blur, set the toplevel `wl_surface`'s blur region to the whole window, updated on resize; a no-op elsewhere, and only while the translucent window is on. Accept when the desktop behind the window shows blurred on GNOME 51 and is unchanged on GNOME 50. Offer it upstream with the GTK4 transparency PR. Windows already blurs through Acrylic.
- A hosted share service with short codes and share versioning (running costs).
- In-app mod search and browsing: Mortar links out to Nexus.
- Browser extension marks Nexus listings and search results: hide or highlight mods the active profile already has, so browsing shows what is missing. The extension would ask Mortar for the active profile's Nexus mod ids over the native messaging host it already uses for `nxm://` links.
- ModDrop as a source (no documented download API); the Xbox app version (WindowsApps folders are locked down).
- Windows code signing.
- macOS, as Stardrop ships for x64 and arm64: needs an Apple developer account for signing and notarization, Mac Steam paths and nxm registration, and a Mac to test on.
- More interface languages than English, as Stardrop (17+), MO2 and r2modman ship; every string already goes through Lingui (English only for v1).
- A light theme, as in Stardrop and Vortex, built once the design system is revisited (MUI stays; Tailwind and shadcn were raised).
- Portable mode: the data folder beside the executable, switched by a marker file (Move data folder exists).
- Previewing an archive's file tree before installing it (the folder picker shows it only when no manifest is found).
- Comparing a profile against a friend's share link or `.mortar` file for multiplayer, then installing what is missing.
- Bottles as a launcher (Linux): games there are Windows builds in a Wine prefix, so it needs SMAPI's Windows installer run inside the bottle (`bottles-cli run -b <bottle> -e <installer>`) and launches through `bottles-cli run` with `--mods-path`; the Linux SMAPI install would break such a copy. Detection is simple: bottles under `~/.local/share/bottles/bottles` and `~/.var/app/com.usebottles.bottles/data/bottles/bottles`, each searched for `drive_c/Program Files (x86)/Steam` and GOG folders.
- CLI verbs for the profile shortcut, Add to Steam and the Steam launch option (`mortar profile shortcut`, `mortar profile steam`, `mortar game steam-launch-option`), matching the GUI actions.
- Choosing a conflict winner: "Make <mod> win" adds an optional dependency on the losing mod to the winner's installed manifest, so SMAPI loads the winner later and its same-priority edits apply last; recorded so updates re-apply it. Load-against-Load conflicts still need one mod switched off.
- Steam Deck Game Mode play: a `--play` launch from a Steam shortcut starts Mortar minimised or headless, shows only a small controller-friendly prompt when Play is blocked, and exits when the game closes so Steam tracks playtime.
- Nexus tracked mods: show which installed mods are tracked, track or untrack from the mod menu, and a "Tracked, not installed" list to pick from (v1 tracked_mods endpoints, signed-in key).
- A profile sync folder (Syncthing, Dropbox, a NAS) holding each profile's `.mortar` state, so another machine is offered the changes, with conflict detection when both sides edited; mod files still come from their sources.
- Per-profile save isolation.
- Registering Mortar with Nexus (SSO slug; ask then about OAuth, which Vortex uses via `nxm://oauth/callback`, and Collections), and a mode for users without an API key: an `nxm://` link cannot become a download without API authentication (HTTP 401 without a key, measured), so that mode would pick up manual downloads from the Downloads folder by their manifests, with confirmation.
