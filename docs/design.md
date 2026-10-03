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

## Settings reorganisation

Decided with NOMAD (Settings pages grew by accretion: per-game prefs on global pages, three groups called "Mods", inline unbounded lists). Code: `frontend/src/settings/SettingsPage.tsx` (section list), `sections/*.tsx`, per-game page `sections/GameSettings.tsx`; pref scope comes from `internal/settings/registry.go`.

- **Pages, in nav order:**
  - General: Startup and window (on Play, start screen, launch at login, start minimised, remember window, keep in tray), Language, Help (tips, tour), Settings file (Export…, Import…).
  - Appearance: Theme, Accent, Background, Display (dates, density, reduce motion, profile hero), Mods view (default view, card size, author on cards, grouping, sort), Sidebar (profile order, badge counts).
  - Mods and profiles: Installing (enable on install, auto-enable requirements, missing requirements, reuse FOMOD choices, confirm removals, modified outside Mortar), Problems (harmless conflicts, scan depth, background badge checks).
  - Downloads: Mod Manager Download links (+ other games' redirect), browser extension, default profile for Nexus links, parallel, auto-retry, pause while playing, verify MD5, preferred server, download folder, keep archives.
  - Nexus account: account banner, Tracking (auto-track, untrack all…, untrack unused…), Endorsements, API quota.
  - Updates: Mortar (status, beta, install automatically), Mods (check on start, interval, enabled only, pre-release versions).
  - Notifications: downloads finished/failed, run crashed, mod-update digest (one control replacing "Notify when updates are found" + "Update digest notification").
  - Storage (replaces Data): Location (path, Open folder, Move…), Usage by game with right-aligned sizes in the app font, summary rows with a size and a button each (Clean up store…, Deleted profiles…, Clear cache), Retention (unused store item days, trash days, history events, run logs kept, save backups kept).
  - Launchers, Shortcuts (group "Console" renamed "Tabs", group names translated), About: unchanged otherwise.
- **Per-game page gets** every game-scope pref now on global pages: default launch, SMAPI console window, enable/missing requirements, harmless conflicts, conflict scan depth, play backups (before Play, kept, location), console level/timestamps/follow/log cap, update mods before Play default, SMAPI unofficial builds. Global page rows stop writing `settings.lastGame || 'stardew'` (`PrefRow.tsx:28`, `DataPrefs.tsx:36`, `SmapiVersionRow.tsx:75`); the game page passes its game.
- **One row pattern:** every control is a `SettingRow` (label and description left, control right) inside a titled `SettingsSection`; no floating-label fields, no `FormControlLabel` switches, no loose text boxes in cards, no untitled or single-row sections where a neighbour fits.
- **No reset links:** "Reset section to defaults" and `ResetSection.tsx` are removed (NOMAD: remove entirely). Shortcuts keeps per-shortcut Reset; "Reset all" gets a confirm.
- **Lists and flows open dialogs:** Store cleanup (unused + duplicates, one list with select-all and a single error-coloured confirm, replacing both the inline list in `DataStoreReport.tsx` and the separate "Clean up unused mod files…" flow), Deleted profiles, Clear cache, Move data folder.
- **Defects to fix on the way:** empty "Default sort" and "Default profile for Nexus links" selects; Nexus tracking text hard-coded "Stardew Valley"; Cache/Trash size misalignment; "0 byte" pluralisation.
- **Done when** every pref in `registry.go` appears exactly once (game scope only on the game page), search finds every row, no page renders an unbounded list inline, and `gui-design.md` describes the new pages.

## Settings visual pass (Steam-style)

Decided with NOMAD after comparing with Steam's Settings. Code: `frontend/src/settings/SettingsSection.tsx` (section + row), `SettingsNav.tsx`, `PrefControls.tsx`, `sections/*`.

- **Rows as tiles:** each `SettingRow` is its own rounded tile (8px gap, `--mortar-overlay-45`), no single card with hairline dividers; section titles are 18px semibold headings in the primary ink.
- **Type:** labels 16px, descriptions 14px secondary and wrap in full (no one-line ellipsis).
- **Nav:** an icon per item (lucide), full-width highlighted selection, dividers between groups: General, Appearance | Mods and profiles, Downloads, Nexus account, Updates, Notifications | Storage, Launchers | Shortcuts, About.
- **Filled controls:** selects and row buttons are filled (`--mortar-raised`) with a chevron, not outlined; the accent colour only on the one primary action of a page or dialog.
- **Segmented buttons** for 2–4 option prefs (theme, dates, density, default view, card size, harmless conflicts, reduce motion, profile hero, sidebar badges); dropdowns stay for longer or dynamic lists.
- **Radio cards** for modes with consequences: When you press Play, Start screen, Backup before Play.
- **Storage like Steam:** inline legend under the bar (dot, LABEL, size); store cleanup is an in-page list (checkboxes, sizes right, newest-copy rule) with Remove anchored at the bottom when something is selected, replacing the cleanup dialog's store part; leftover files stay a row with Clear….
- **Notification matrix:** one row per event (download finished, download failed, run crashed, mod updates found) with columns In-app toast and Desktop notification; adds a desktop-notification channel (Wails notifications on Linux/Windows) and per-event, per-channel prefs.
- **Done when** every Settings page uses only these patterns and NOMAD signs off on screenshots.

## Queued for v1

- **Library:** an extra folder to scan for mods; a toggle to show dot-hidden mods; asking before deleting old files on update; new folders in the game's own `Mods` folder offered for moving into a profile.
- **Packages:** an aarch64 Flatpak bundle is blocked without qemu binfmt (or an aarch64 host): `flatpak-builder --arch=aarch64` still runs `build-commands` via the aarch64 SDK's `/bin/sh` (`bwrap: execvp /bin/sh: Exec format error`), even with only `install` of a prebuilt binary.
- **Self-test:** update a mod in place with an older GitHub release in a copied game folder, checking carry-over, `.mortar-old`, the save backup and Roll back.
- **CurseForge** as a third source, after the repository is public: apply for a 3rd-party API key, then build it without caching API data, with a User-Agent on every request, and honouring each author's distribution setting.

## Later

- **UI translations** beyond English: the Lingui machinery and extracted catalogs exist; needs chosen languages and translators. Parked 2026-10-02 (not v1).
- **Steam Deck / gamepad mode** (larger targets, gamepad focus navigation, Game Mode): parked 2026-10-02 (not v1).
- **macOS build**: Stardew runs on macOS, but Mortar has no macOS CI or test machine. Parked 2026-10-02 (not v1).
- **Scheduled save backups** (daily or every N hours while Mortar runs, keep the last N per save, never while the game writes), beside today's before-Play backups: parked 2026-10-03.
- **Accessibility pass** (keyboard-only navigation of every screen, focus order, screen-reader labels on icon buttons, reduced motion everywhere): parked 2026-10-03.
- **Offline mode banner** (clear banner when Nexus/GitHub are unreachable, cached data with "as of" times, network actions disabled with a reason): parked 2026-10-03.

Not in the first release; re-weigh only when asked:

- Game Select hover in the style of Concrete's switcher, where the hovered game grows while the others become strips, with parallax and brightness shifts: eased `flex-grow` and `flex-basis` looked jumpy in WebKitGTK, so v1 has no hover effect; revisit with a measured, transform-based approach.
- Lethal Company, as the second `Game` implementation ([lethal-company.md](lethal-company.md)).
- Translucent window, desktop showing through; on Windows it needs Acrylic measured with the frameless window. The see-through window looked wrong, so v1 is solid. The `Rethunk-AI/wails` fork's GTK4 `setTransparent()` fix (upstream wailsapp/wails#6197) makes it possible: stock GTK4 leaves `setTransparent()` empty (`wailsapp/wails` `v3/pkg/application/linux_cgo.go:1418`), and the fix registers a display-wide CSS provider that clears the window background except the title bar. Any fading or `backdrop-filter` full-window layer turns WebKitGTK's translucent window opaque; a static tint does not. Stacked alphas compound toward opaque, so images and overlays each need their own alpha. Frosted glass needs `ext-background-effect-v1`, below.
- Frosted glass on Linux, built when the desktop runs GNOME 51. CSS cannot do it: `backdrop-filter` sees only the webview's pixels, and a full-window filter layer turns WebKitGTK's translucent window opaque. The compositor blurs behind the window through the Wayland protocol `ext-background-effect-v1` (Mutter from GNOME 51, KWin from Plasma 6.7; GTK 4.23.3 speaks it). Shape: in the `Rethunk-AI/wails` fork, beside `setTransparent()` in `v3/pkg/application/linux_cgo.go`, bind `ext_background_effect_manager_v1` on Wayland, and when it advertises blur, set the toplevel `wl_surface`'s blur region to the whole window, updated on resize; a no-op elsewhere, and only while the translucent window is on. Accept when the desktop behind the window shows blurred on GNOME 51 and is unchanged on GNOME 50. Offer it upstream with the GTK4 transparency PR. Windows already blurs through Acrylic.
- A hosted share service with short codes and share versioning (running costs).
- In-app mod search and browsing: Mortar links out to Nexus.
- ModDrop as a source (no documented download API); the Xbox app version (WindowsApps folders are locked down).
- Windows code signing.
- macOS, as Stardrop ships for x64 and arm64: needs an Apple developer account for signing and notarization, Mac Steam paths and nxm registration, and a Mac to test on.
- More interface languages than English, as Stardrop (17+), MO2 and r2modman ship; every string already goes through Lingui (English only for v1).
- Portable mode: the data folder beside the executable, switched by a marker file (Move data folder exists).
- Previewing an archive's file tree before installing it (the folder picker shows it only when no manifest is found).
- Bottles as a launcher (Linux): games there are Windows builds in a Wine prefix, so it needs SMAPI's Windows installer run inside the bottle (`bottles-cli run -b <bottle> -e <installer>`) and launches through `bottles-cli run` with `--mods-path`; the Linux SMAPI install would break such a copy. Detection is simple: bottles under `~/.local/share/bottles/bottles` and `~/.var/app/com.usebottles.bottles/data/bottles/bottles`, each searched for `drive_c/Program Files (x86)/Steam` and GOG folders.
- Steam Deck Game Mode play: a `--play` launch from a Steam shortcut starts Mortar minimised or headless, shows only a small controller-friendly prompt when Play is blocked, and exits when the game closes so Steam tracks playtime.
- A profile sync folder (Syncthing, Dropbox, a NAS) holding each profile's `.mortar` state, so another machine is offered the changes, with conflict detection when both sides edited; mod files still come from their sources.
- Needs Nexus's approval through app registration first (below), since it starts downloads outside Nexus's own Mod Manager Download button: an "Add to Mortar" button on Nexus listing tiles.
- Profile templates: a new profile started from a bundle plus game settings and launch options.
- Per-profile save isolation.
- Settings considered and not taken (2026-10-02): new profiles starting as a copy of the open profile or from a bundle; scheduled save backups on a timer while the game runs; an offline mode that never contacts the network.
- Registering Mortar with Nexus (SSO slug; ask then about OAuth, which Vortex uses via `nxm://oauth/callback`, and Collections).
