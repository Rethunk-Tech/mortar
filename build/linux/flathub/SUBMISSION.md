# Flathub submission for Mortar @VERSION@

The owner opens this PR; nothing here is automated. `tech.rethunk.Mortar.yml` beside this file is rendered
by the release workflow with the real sha256 of every source. The release attaches it as `flathub-tech.rethunk.Mortar.yml`.

## Steps

1. Fork https://github.com/flathub/flathub and branch from `new-pr` (not `master`).
2. Add the rendered manifest to the branch as `tech.rethunk.Mortar.yml` (rename the release asset), nothing else.
3. Open the PR against `flathub/flathub`'s `new-pr` branch with the title and body below.
4. Answer the review bot's lint output; once merged, Flathub creates `flathub/tech.rethunk.Mortar`, which the
   owner maintains from then on (bump `url`/`sha256` per release).

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
https://github.com/Rethunk-AI/mortar (AGPL-3.0-only)

### Checklist
- [x] The manifest builds from release URLs with sha256 (no local paths)
- [x] AppStream metainfo, desktop file, 256 and 512 px icons and the scalable icon are installed
- [x] The id `tech.rethunk.Mortar` matches a domain the developer controls (rethunk.tech)
- [x] The license text and third-party notices are installed under /app/share/doc/mortar

### Notes for reviewers
- The binary is the upstream release `mortar-linux-amd64` (Go, built in CI from the tagged commit), installed
  as a prebuilt file; a from-source Go build is not offered because the Wails fork is pinned through go.mod
  `replace`. x86_64 only.
- finish-args: network (mod downloads), Wayland/X11 and DRI (WebKitGTK window), and filesystem access to the
  game libraries it manages: native and Flatpak Steam, Heroic and Lutris configs (read-only), Flatpak Lutris and
  Heroic app data, ~/GOG Games and ~/Games/Heroic, plus its own xdg-data/mortar. These are needed to find and
  launch the installed game and write mods into its folder.
- `--own-name=tech.rethunk.mortar.SingleInstance` keeps a second launch from opening a second window.
```
