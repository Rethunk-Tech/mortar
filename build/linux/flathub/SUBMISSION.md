# Flathub submission for Mortar @VERSION@

Owner-only PR. Flathub forbids AI agents on submission PRs and AI-assisted content in the submitted manifest; the owner writes that manifest, records the video, completes the checklist (including AI-generated code in Mortar), and waits for a meaningful public history. `build/linux/flathub/render.sh` attaches `flathub-` prefixed files as reference, never as the submission: `tech.rethunk.Mortar.yml` (v@VERSION@ from source), `go-sources.json`, `node-sources.json`, `yarn.lock`. finish-args come from `build/linux/flatpak/tech.rethunk.Mortar.yml`.

## Steps

1. Fork https://github.com/flathub/flathub and branch from `new-pr` (not `master`).
2. Add the four files under their names without the `flathub-` prefix, nothing else.
3. Open the PR against `flathub/flathub`'s `new-pr` with the title and body below.
4. Answer the review bot. After merge, Flathub creates `flathub/tech.rethunk.Mortar`; per release, replace the four files.

## PR title

Add tech.rethunk.Mortar

## PR body

```
### App name
Mortar

### App id
tech.rethunk.Mortar

### Description
Desktop mod manager for several games (Stardew Valley, Lethal Company, Valheim, Vintage Story, The Sims 4 and many
Thunderstore games). Finds the game, installs its mod loader, keeps each set of mods in a profile, launches the game
with the chosen profile and shares a profile as a link.

### Homepage / source
https://github.com/Rethunk-Tech/mortar (AGPL-3.0-only)

### Checklist
- [x] The manifest builds from source at a tagged commit, offline, from public URLs with checksums (no local paths)
- [x] AppStream metainfo, desktop file, 256 and 512 px icons and the scalable icon are installed
- [x] The id `tech.rethunk.Mortar` matches a domain the developer controls (rethunk.tech)
- [x] The license text and third-party notices are installed under /app/share/doc/mortar

### Notes for reviewers
- Built from source: Go with org.freedesktop.Sdk.Extension.golang from proxy.golang.org module files served to
  the build through `GOPROXY=file://` (go-sources.json), and the React frontend with
  org.freedesktop.Sdk.Extension.node24 and yarn's offline mirror (node-sources.json, from flatpak-node-generator;
  the project uses bun, so `yarn.lock` is bun's lock written in yarn's format). The Wails CLI is built from the
  same Go sources to generate the frontend's TypeScript bindings. No install scripts run.
- finish-args: network (mod downloads), Wayland/X11 with ipc and DRI (WebKitGTK window), input devices (controller
  navigation), xdg-download (mod archives the user downloads), `~/GOG Games` and `~/Games/Heroic` (default GOG and
  Heroic install folders), /run/media and /mnt (game libraries on other drives), org.freedesktop.Notifications (download and run notices) and org.kde.StatusNotifierWatcher with
  `org.kde.StatusNotifierItem-2-1` (the optional tray icon). Account keys are sealed with the Secret portal's key, launch at login uses the Background portal, and per-profile shortcuts the DynamicLauncher portal; the
  single-instance name is `tech.rethunk.Mortar.SingleInstance`, under the app id.

### Linter exceptions requested
Mortar is a mod manager: it finds a game another launcher installed, writes mods into that game's folder and starts
the game through that launcher. Each exception below is the game library or launcher state it manages:
- `finish-args-unnecessary-xdg-data-Steam-rw-access`, `finish-args-flatpak-appdata-folder-com.valvesoftware.Steam-rw-access`:
  native and Flatpak Steam libraries, where Stardew Valley is installed and its Mods folder lives.
- `finish-args-unnecessary-xdg-config-StardewValley-rw-access`, `finish-args-flatpak-appdata-folder-com.valvesoftware.Steam-.config-StardewValley-rw-access`:
  the game's save folders (native and Flatpak Steam), for save backups and the per-save mod list.
- `finish-args-unnecessary-xdg-config-heroic-ro-access`, `finish-args-unnecessary-xdg-data-lutris-ro-access`:
  read-only Heroic and Lutris configs, to find a GOG install those launchers manage.
- `finish-args-flatpak-appdata-folder-net.lutris.Lutris-rw-access`, `finish-args-flatpak-appdata-folder-com.heroicgameslauncher.hgl-rw-access`:
  the same for Flatpak Lutris and Heroic, whose game folders live in their app data.
- `finish-args-unnecessary-xdg-data-bottles-rw-access`, `finish-args-flatpak-appdata-folder-com.usebottles.bottles-rw-access`:
  native and Flatpak Bottles bottles, where a Windows build of the game lives and SMAPI's Windows installer writes.
- `finish-args-flatpak-spawn-access`: `flatpak-spawn --host` starts the game through the user's host Steam or
  launcher, which cannot run inside this sandbox; there is no portal for starting another launcher's game.
```
