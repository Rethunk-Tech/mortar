# Flathub submission for Mortar @VERSION@

The owner opens this PR; nothing here is automated. Flathub's Generative AI policy forbids AI agents opening, writing or
replying to submission PRs and forbids AI-assisted content in the manifest, so the owner writes the manifest that is
submitted, records the required video of the Flatpak running, and completes the template checklist, including the
disclosure of AI-generated code in Mortar. Flathub also expects a meaningful public development history, so this
submission waits until the public repository has one. The release workflow renders the submission with
`build/linux/flathub/render.sh` and attaches its files with a `flathub-` prefix: `tech.rethunk.Mortar.yml`
(builds the v@VERSION@ tag from source), `go-sources.json` and `node-sources.json` (every Go module and npm
package the build reads, with sha256 or sha512) and `yarn.lock`. Those rendered files, finish-args included (they are
copied from the sideload manifest, `build/linux/flatpak/tech.rethunk.Mortar.yml`), were written with AI help: they are the
owner's reference, never what is submitted.

## Steps

1. Fork https://github.com/flathub/flathub and branch from `new-pr` (not `master`).
2. Add the four files to the branch under their names without the `flathub-` prefix, nothing else.
3. Open the PR against `flathub/flathub`'s `new-pr` branch with the title and body below.
4. Answer the review bot's lint output; once merged, Flathub creates `flathub/tech.rethunk.Mortar`, which the
   owner maintains from then on (per release, replace the four files with the new release's).

## PR title

Add tech.rethunk.Mortar

## PR body

```
### App name
Mortar

### App id
tech.rethunk.Mortar

### Description
Desktop mod manager for Stardew Valley (SMAPI, Nexus Mods). Finds the game, installs its mod loader, keeps each
set of mods in a profile, launches the game with the chosen profile and shares a profile as a link.

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
- finish-args: network (mod downloads), Wayland/X11 with ipc and DRI (WebKitGTK window), xdg-download (mod archives
  the user downloads), /run/media and /mnt (game libraries on other drives), org.freedesktop.secrets (the Nexus
  sign-in), org.freedesktop.Notifications (download and run notices) and org.kde.StatusNotifierWatcher with
  `org.kde.StatusNotifierItem-2-1` (the optional tray icon). Launch at login uses the Background portal and per-profile shortcuts the DynamicLauncher portal; the
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
