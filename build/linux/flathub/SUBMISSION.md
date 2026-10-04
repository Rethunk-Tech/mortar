# Flathub submission for Mortar @VERSION@

The owner opens this PR; nothing here is automated. The release workflow renders the submission with
`build/linux/flathub/render.sh` and attaches its files with a `flathub-` prefix: `tech.rethunk.Mortar.yml`
(builds the v@VERSION@ tag from source), `go-sources.json` and `node-sources.json` (every Go module and npm
package the build reads, with sha256 or sha512) and `yarn.lock`.

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
- finish-args: network (mod downloads), Wayland/X11 and DRI (WebKitGTK window), and filesystem access to the
  game libraries it manages: native and Flatpak Steam, Heroic and Lutris configs (read-only), Flatpak Lutris and
  Heroic app data, ~/GOG Games and ~/Games/Heroic, plus its own xdg-data/mortar. These are needed to find and
  launch the installed game and write mods into its folder. Also xdg-download (mod archives the user downloads),
  /run/media and /mnt (game libraries on other drives), xdg-config/autostart and xdg-data/applications (launch at
  login and per-profile desktop shortcuts), org.freedesktop.secrets (the Nexus sign-in) and
  org.freedesktop.Flatpak (`flatpak-spawn --host` to start Steam and the game outside the sandbox).
- `--own-name=tech.rethunk.mortar.SingleInstance` keeps a second launch from opening a second window.
```
