# Mortar: decided, not yet built

This file holds only work that is decided but not built. Each item is written to be implemented cold: the shape, the evidence it rests on, the traps, and when it is done. When an item lands, delete it here and move any fact that stays true to [architecture.md](architecture.md); the code is the record of what exists. Screens and styling: [gui-design.md](gui-design.md). Lethal Company research for the second game: [lethal-company.md](lethal-company.md). Standing rules: [AGENTS.md](../AGENTS.md).

Items marked **Measure** need a throwaway test first; those tests run outside this repo and only their results land here. Measurements were taken as stated in architecture.md.

## Sharing: the static page deploy

Remaining: deploying `site/stardew/p/` and `site/download/` at `https://mortar.rethunk.tech/stardew/p` and `https://mortar.rethunk.tech/download/`, held back on purpose until the first release is ready (NOMAD, 2026-09-30). How sharing works: [architecture.md](architecture.md#sharing).

- **Deploy:** `site/stardew/p/index.html` and `site/download/index.html` in this repo, as a DigitalOcean App Platform static site (free tier: three static apps); `rethunk.tech` is on DigitalOcean's nameservers, and `maitre.rethunk.tech` is already a CNAME to an App Platform app, so the subdomain is set up the same way. The share page shows two buttons, since it cannot tell whether a scheme handler exists: open in Mortar (`mortar://stardew/p/<payload>`) and Get Mortar (`/download/`), which also copies the link so the importer can take it after installing.
- **Done when** the link opens the page on the live domain, its button opens Mortar's Import with the link filled in, and no request the page makes carries the fragment.

## Release

Remaining ([architecture.md](architecture.md#release)):

- The updater fixes are offered upstream as wailsapp/wails#6200 (EXDEV staging), #6201 (AppImage) and #6202 (OnUpdateApplied, draft pending a WEP); Mortar pins the fork until they ship in a tagged v3 beta; then pin that beta and drop the `Rethunk-AI/wails` replace in `go.mod`. wailsapp/wails#6197 (GTK4 transparency) does not gate this: it only serves the translucent window below.
- The repo turns public at the first release and builds go on its GitHub Releases, since the updater's manifest and assets must be publicly downloadable.
- **Measure on Windows:** how launch arguments order around `%command%`, and whether SMAPI needs `--no-terminal`; one real update through the updater, and `DisplayVersion` after it.

## Queued for v1

Decided with NOMAD on 2026-10-01 from the competitor review (Vortex, MO2, Stardrop, r2modman, Gale); each is one landable unit.

- **Installs:** multi-file installs from one Nexus page as one entry; variant folders in one archive (sibling folders with the same `UniqueID`) ask which variant, remembered per entry; queue downloads (Nexus, GitHub) open the root picker the way dropped archives do; shares and `.mortar` files carry FOMOD choices and the chosen archive root and replay them on import; Update review re-offers FOMOD options when an update's config changes them.
- **Problems:** the last run's erroring mods as problems with Switch off and Get help; Content Patcher conflicts name the winning pack (`Priority` Early, Default, Late, then load order); SMAPI's abandoned status with its suggested replacement and Add; the outside-edit snapshot refreshed after install, update, rollback and import, and outside edits counted in the problem badge and the Problems status group; a crash bisect helper that halves the mods in a temporary copy of the profile and relaunches until one culprit is left.
- **Updates:** a pre-release toggle, checking only enabled mods, an hourly re-check while Mortar runs, and enabling mods on install as a setting; the author hints `UpdateCautionMessage` and `DeleteOldVersion`; Mortar's own update downloaded in the background and applied on quit, then What's new once.
- **Library:** custom categories with a primary category per mod; a preferred Nexus download server; per-game nxm redirect to the previous handler; an extra folder to scan for mods, a toggle to show dot-hidden mods, and asking before deleting old files on update; import from Stardrop and Vortex keeps each mod's `config.json`; new folders in the game's own `Mods` folder offered for moving into a profile.
- **Tray:** Play for recent profiles, the game's running state, and a notification with the crash summary when a run ends.
- **Packages:** an aarch64 Flatpak bundle, if `flatpak-builder` can build it without a registered emulator; the Flatpak granted read access to GOG, Heroic and Lutris install locations so those stores are found inside the sandbox.
- **Self-test:** update a mod in place with an older GitHub release in a copied game folder, checking carry-over, `.mortar-old`, the save backup and Roll back.
- **CurseForge** as a third source, after the repository is public: apply for a 3rd-party API key with the application text NOMAD approved, then build it without caching API data, with a User-Agent on every request, and honouring each author's distribution setting.

## Later

Not in the first release, each by NOMAD on 2026-09-29; re-weigh only when asked:

- Game Select hover in the style of Concrete's switcher, where the hovered game grows while the others become strips, with parallax and brightness shifts (NOMAD, 2026-09-30): eased `flex-grow` and `flex-basis` looked jumpy in WebKitGTK, so v1 has no hover effect; revisit with a measured, transform-based approach.
- Lethal Company, as the second `Game` implementation ([lethal-company.md](lethal-company.md)).
- Translucent window, desktop showing through (NOMAD, 2026-09-30, deferred to v2); on Windows it needs Acrylic measured with the frameless window. The see-through window looked wrong, so v1 is solid. The `Rethunk-AI/wails` fork's GTK4 `setTransparent()` fix (upstream wailsapp/wails#6197) makes it possible: stock GTK4 leaves `setTransparent()` empty (`wailsapp/wails` `v3/pkg/application/linux_cgo.go:1418`), and the fix registers a display-wide CSS provider that clears the window background except the title bar. Any fading or `backdrop-filter` full-window layer turns WebKitGTK's translucent window opaque; a static tint does not. Stacked alphas compound toward opaque, so images and overlays each need their own alpha. Frosted glass needs `ext-background-effect-v1`, below.
- Frosted glass on Linux (NOMAD, 2026-09-30), built when NOMAD's desktop runs GNOME 51. CSS cannot do it: `backdrop-filter` sees only the webview's pixels, and a full-window filter layer turns WebKitGTK's translucent window opaque. The compositor blurs behind the window through the Wayland protocol `ext-background-effect-v1` (Mutter from GNOME 51, KWin from Plasma 6.7; GTK 4.23.3 speaks it). Shape: in the `Rethunk-AI/wails` fork, beside `setTransparent()` in `v3/pkg/application/linux_cgo.go`, bind `ext_background_effect_manager_v1` on Wayland, and when it advertises blur, set the toplevel `wl_surface`'s blur region to the whole window, updated on resize; a no-op elsewhere, and only while the translucent window is on. Accept when the desktop behind the window shows blurred on GNOME 51 and is unchanged on GNOME 50. Offer it upstream with the GTK4 transparency PR. Windows already blurs through Acrylic.
- A hosted share service with short codes and share versioning (running costs).
- In-app mod search and browsing: Mortar links out to Nexus.
- ModDrop as a source (no documented download API); the Xbox app version (WindowsApps folders are locked down).
- Windows code signing.
- macOS, as Stardrop ships for x64 and arm64: needs an Apple developer account for signing and notarization, Mac Steam paths and nxm registration, and a Mac to test on (NOMAD, 2026-10-01).
- More interface languages than English, as Stardrop (17+), MO2 and r2modman ship; every string already goes through Lingui (NOMAD, 2026-10-01: English only for v1).
- A light theme, as in Stardrop and Vortex, built once the design system is revisited (MUI stays for now; Tailwind and shadcn were raised) (NOMAD, 2026-10-01).
- Per-profile save isolation.
- Registering Mortar with Nexus (SSO slug; ask then about OAuth, which Vortex uses via `nxm://oauth/callback`, and Collections), and a mode for users without an API key: an `nxm://` link cannot become a download without API authentication (HTTP 401 without a key, measured), so that mode would pick up manual downloads from the Downloads folder by their manifests, with confirmation.
