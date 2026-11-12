# TODO

Work that is decided and not yet built. `AGENTS.md` and `docs/` describe what exists;
this file is the only place that describes what does not. Delete an entry in the
change that lands it. Design text lives in [docs/design.md](docs/design.md).

## Mortar pins a Wails fork instead of an upstream release

`go.mod` still `replace`s Wails with `Rethunk-AI/wails` branch `mortar/v3.1.0-combined`
(tags `v3.1.0-mortar.N`).
Every Mortar build, the AppImage path and the GTK4 window rest on that fork, so
an upstream Wails v3 tag cannot be taken until it carries the same hooks. The
cost is a private pin that has to be rebased onto each Wails move, and a
release that cannot yet drop the replace.

The decided shape is to drop the replace and pin an upstream release once one
carries what the branch adds. Rejected: shipping Mortar against stock Wails
before those PRs land, or carrying a permanent fork.

### Read

- `go.mod` — the `Rethunk-AI/wails` replace and the pin this change drops.
- [docs/architecture.md](docs/architecture.md#stack) — the stack pin this
  release is allowed to move.
- [docs/design.md](docs/design.md#release) — the WEP/PR list that gates the drop.

### Traps

- **errorkind falls back through `Options.MarshalError`.** Upstream #6199; Mortar's
  `errorkind_test.go` relies on service methods using that fallback. A pin that
  lands without it fails those tests with a different wire error.
- **Single-instance lock released at shutdown start.** Upstream #6240. Holding
  the lock until process exit blocks the next instance during an update restart.
- **Apply-on-exit is fork-only.** No upstream WEP yet. Dropping the replace
  before that exists removes the apply-on-exit path Mortar already ships.

### Acceptance

- `go.mod` has no `Rethunk-AI/wails` replace and pins an upstream Wails module
  version.
- That version includes #6202 (OnUpdateApplied, WEP #6203), #6199, #6240, #6197,
  #6281, #6282, #6283, #6526, #6571 to #6578, #6582, WEPs #6243, #6244 and #6580,
  apply-on-exit and the server-mode shutdown hooks.
- `bun run gate` stays clean.

## The Linux game sandbox is unproven on real games

A malicious mod runs in the game process and can read `~/.ssh`, browser
profiles and Mortar's data folder. The bubblewrap path is built and opt-in
(maintainer decision of 2026-11-07); what has not been proven is that a real
game launch hides the home canaries, still writes the game's own folders, and
loads every enabled mod. Until that run exists, the setting is a claim, not a
measured control.

The decided shape is the existing opt-in sandbox (`sandboxLaunch`,
`sandboxExtraFolders`), proven on copied game folders. Rejected until asked:
seccomp, a network namespace with a loopback-only bridge, Windows AppContainer
and macOS.

### Read

- `internal/sandbox/sandbox.go:70` (`sandbox.Argv`) — bubblewrap argv and mounts.
- `internal/sandbox/sandbox.go:138` (`sandbox.Credentials`) — credential-looking
  environment variables stripped.
- `internal/sandbox/sandbox.go:155` (`sandbox.SteamRO`) — Steam parts Proton
  reads, bind-mounted read-only.
- `internal/sandbox/dbus.go:35` (`sandbox.ProxyArgv`) and
  `internal/sandbox/dbus.go:81` (`sandbox.StartProxy`) — filtered session bus.
- `internal/sandbox/dbus.go:120` (`sandbox.TaskCap`) — process cap.
- `internal/sandbox/folders.go:12` (`sandbox.ExtraFolders`),
  `folders.go:38` (`sandbox.Allowable`), `folders.go:60` (`sandbox.holdsSecrets`)
  — extra folders and the refusals for keys and sign-ins.
- `internal/game/start.go:296` — `game.sandboxed` applies it to a direct start
  and fails instead of starting unsandboxed (`sandbox.ErrNoBwrap` at
  `sandbox.go:18`).
- `internal/launchsvc/sandbox.go:18` (`launchsvc.sandboxSpec`) — profile, saves,
  cache and extra folders.
- `internal/launchsvc/service.go:1102` — the setting forces a direct start even
  where the store would relay.
- `internal/launchsvc/runs.go:63` (`Run.Sandbox`) — folders the run could write.
- `internal/launchsvc/sandbox.go:99` (`launchsvc.SandboxReport`) and
  `internal/sandbox/denial.go:20` (`sandbox.Denials`).
- `frontend/src/console/SandboxNote.tsx` and render case `console-sandboxed-run`
  (`frontend/e2e/stateCases.ts:340`).
- `internal/settings/registry.go:271` — `sandboxLaunch` and `sandboxExtraFolders`.
- `frontend/src/settings/GameSettingsPage.tsx:84` and
  `frontend/src/settings/prefCopyGame.ts:130`.
- `scripts/selftest.sh` `regress_sandbox_check` — `MORTAR_REGRESS_SANDBOX`
  reaches the Stardew and Lethal Company regress; neither has run sandboxed on
  a real game yet.
- `internal/sandbox/sandbox_test.go`
  `TestSandboxedProcessSeesNeitherSSHNorARealHomeCanary` — real bwrap, no game.
- Tests: `internal/sandbox/sandbox_test.go`,
  `internal/launchsvc/sandbox_spec_test.go`.

### Traps

- **Never prove this on a real install.** The SMAPI installer writes through
  symlinks; copied game folders only, stopped by the recorded PID.
- **Missing bwrap must fail the launch.** Falling back to unsandboxed is the
  defect `ErrNoBwrap` exists to catch, including when Mortar is the Flatpak.
- **Home must be hidden before extra binds.** The unit tests require that order;
  reversing it lets a canary under `$HOME` show up after an extra-folder bind.

### Acceptance

- A launch prefix that reads `$HOME/.ssh` and a canary file in the real home
  starts inside the sandbox and finds neither, while the game's own folders
  write.
- Stardew with SMAPI starts sandboxed, loads every enabled mod and saves into
  the profile or the Saves folder; the game folder hashes as the plain regress
  requires (`MORTAR_REGRESS_SANDBOX` covers Stardew, not only Lethal Company).
- A gamepad and audio work in a manual run (human-only).
- `bun run gate` stays clean. No seccomp, network namespace, Windows or macOS
  sandbox ships in this change.

## Darkest Dungeon's native Linux build is not in the catalog

Steam launches `_linux/darkest.bin.x86_64` for the 64-bit Linux build (Steam app info for `262060`, `config.launch`; [docs/games.md](docs/games.md#darkest-dungeon)). `GameInfo.LinuxMarker` holds a file name that `GameInfo.Validate` rejects with a path separator, so the entry cannot name it and a native Linux install is read as the Windows build or missed.

The decided shape is `LinuxMarker` as a slash path below the install, validated with `unsafeRel` like `DarwinMarker`, then `linuxMarker` and a `linux` mods path (`{install}/mods`) on the Darkest Dungeon entry. Rejected: a second field for subfolder Linux executables.

### Read

- `internal/components/components.go` — `LinuxMarker` and the file-name check in `Validate` (the "unsafe file or folder name" error).
- `internal/game/install.go` `platformOf`, `internal/game/start.go` `directExe` and `internal/gamestore/drivers.go` (Lutris marker loop) — every reader of `LinuxMarker`.

### Traps

- **`LinuxMarker` also feeds `GameProcesses`** (`internal/game/catalog_only.go`): a path there must reduce to its base name.
- **Unverified GOG folder.** `Darkest Dungeon` is Minigalaxy's name for the folder; no source names the one GOG's own installer makes.

### Acceptance

- A Steam install holding only `_linux/darkest.bin.x86_64` is a native Linux install; one holding `_windows/win64/Darkest.exe` stays the Windows build.
- `scripts/catalog-compat.sh` decodes the catalog in the last two releases.

## Catalog darwin markers are missing for bundles Steam names only as a .app

Steam's app info gives the inner executable for the games with a `darwinMarker` ([docs/games.md](docs/games.md#macos-markers)). Games whose macOS launch entry is just `<Name>.app`, and the layouts where the install folder is the bundle (Dome Keeper, Sun Haven), have none: the executable name is in the bundle's `Info.plist` (`CFBundleExecutable`), which only a Mac install shows. SMAPI's launch behaviour on macOS rests on its installer source, not a Mac run.

The decided shape is to add each missing `darwinMarker` from a Mac install's `Info.plist`, and to let BepInEx's macOS placement read a marker that starts at `Contents/` (`contributeDarwin` in `internal/loader/bepinex5/driver.go`, which needs a first segment ending `.app`). Rejected: guessing the executable from the `.app` name, or treating this entry as the full macOS ship.

### Read

- `internal/loader/bepinex5/driver.go` `contributeDarwin` — the `.app` first-segment rule.
- [docs/design.md](docs/design.md#from-later-picked-2026-11-01) — what is left after the Linux-testable parts.

### Traps

- **A `.app` name is not its executable's name** (`TotallyAccurateBattleSimulator.app` runs `Totally Accurate Battle Simulator`, `MacPlayer.app` runs `Project Gorgon`).

### Acceptance

- Each native-macOS catalog game that should locate on darwin has a cited `darwinMarker`.
- `bun run gate` stays clean. This change does not ship a `.dmg` or notarisation.

## Known-broken Lethal Company content loaders still point at the old packages

Mortar's content-pack side is built (`internal/devsvc/content.go`, the
catalog's `devTemplate.content`) and the loaders are on Thunderstore as
Mortar's Posters, Boombox and Television. New installs of the known-broken
redirects (LethalPosters, LethalPaintings, Custom Boombox Music, TVLoader)
still resolve to the old packages until mortar-known-broken is updated, and
the catalog still carries per-kind `assetKinds` for posters, paintings,
television, boombox, suits and sounds that should leave once no supported
build predates the `content` entry's `minVersion` (`0.4.0`).

The decided shape: those redirects live in Rethunk-Tech/mortar-known-broken
(the list Mortar reads through `internal/components/knownbroken.go`), not in
this repo — LethalPosters and LethalPaintings to `Mortar-Posters`, Custom
Boombox Music to `Mortar-Boombox`, TVLoader to `Mortar-Television`. The
`assetKinds` entries leave the catalog once the oldest supported tag is 0.4.0
or later. Rejected: putting the redirects in this repo, or dropping
`assetKinds` while a supported Mortar older than 0.4.0 still reads the
catalog.

### Read

- `internal/devsvc/content.go` — Mortar's built content-pack side.
- `internal/devsvc/content.go:83` (`contentRecommends`) — New mod recommends
  the folder's loader.
- `internal/devsvc/content.go:206` — a release flags a pack whose manifest
  does not depend on the loader.
- `internal/components/components.json:393`, `:417`, `:425` — catalog names
  for each content folder's `dependency`.
- `internal/components/knownbroken.go` — how Mortar reads
  mortar-known-broken.

### Traps

- **Redirects do not land in this repo.** Editing `components.json` here for
  LethalPosters and friends leaves the list Mortar actually reads unchanged.
- **Dropping `assetKinds` before 0.4.0 is the oldest tag** breaks every still-
  supported build that predates `content`.

### Acceptance

- A pack with posters + boombox + tv in `plugins/mortar/`, suits in
  `plugins/moresuits/` and a sound pack loads all three in game; a
  legacy-layout pack still loads; a code project's ticked content ships in
  its package.
- mortar-known-broken redirects LethalPosters and LethalPaintings to
  `Mortar-Posters`, Custom Boombox Music to `Mortar-Boombox` and TVLoader to
  `Mortar-Television`.
- `assetKinds` for those kinds are gone from the catalog only once the oldest
  supported tag is 0.4.0 or later.
- `bun run gate` stays clean.

## Scrap items cannot be authored without the Unity Editor

A Lethal Company scrap bundle cannot be built outside the Unity Editor:
AssetsTools.NET and UnityPy only patch existing bundles, and a new
MonoBehaviour's script hash cannot be generated. Authors who want a scrap
item are stuck in Unity, so Mortar's wizard cannot offer that kind.

The decided shape is a runtime loader mod in mortar-mods on LethalLib:
`NetworkPrefabs.CloneNetworkPrefab` (deterministic hash from assembly and
name) of template prefabs built once in Unity, an `Item` from
`ScriptableObject.CreateInstance` passed to `Items.RegisterScrap`, filled
from a folder (`item.json`, `texture.png`, `icon.png`, sounds). Mortar's
wizard gains a Scrap item kind that writes the folder. Sholiver-
LethalCardLoader proves the PNG+JSON half. Arbitrary GLB meshes (glTFast
with an HDRP material mapper copied from a vanilla scrap) come after and
were rejected for the first ship: they risk the game's Burst/Collections
versions and grip offsets.

### Read

- [docs/design.md](docs/design.md#later) — the LethalLib path and the GLB
  deferral.
- Mortar's New mod / content wizard (the kind list this Scrap item kind
  joins) and [docs/games.md](docs/games.md#lethal-company).

### Traps

- **Do not generate a new MonoBehaviour script hash outside Unity.** That is
  the reason the bundle path was rejected.
- **GLB meshes are not part of this entry.** glTFast plus an HDRP mapper
  fights Burst/Collections versions and grip offsets; they come after a
  folder-based scrap ships.

### Acceptance

- A folder (`item.json`, `texture.png`, `icon.png`, sounds) written by Mortar's
  Scrap item kind loads in Lethal Company through the LethalLib runtime loader
  without a Unity Editor pass.
- The first ship has no arbitrary GLB mesh path.
- `bun run gate` stays clean.

## The window cannot be translucent

The main window is a solid wallpaper-plus-tint surface. Upstream Wails #6197
(the fork's GTK4 `setTransparent()` fix) is the decided way to make it
translucent; until that is on and checked, the frameless window cannot show
desktop content behind it.

The decided shape is translucent through that GTK4 fix. Windows needs Acrylic
with the frameless window. The AppImage is the GTK 3 build (RGBA visual) and
needs the same check. Rejected: a fading or `backdrop-filter` full-window
layer, which turns WebKitGTK's translucent window opaque.

### Read

- [docs/design.md](docs/design.md#later) — the #6197 pin and the AppImage GTK 3
  check.
- [docs/gui-design.md](docs/gui-design.md) — screens and styling this change
  must still match.
- The fork's GTK4 `setTransparent()` path (upstream #6197).

### Traps

- **A fading or `backdrop-filter` full-window layer turns WebKitGTK's
  translucent window opaque.** Stacked alphas compound toward opaque, so
  images and overlays each need their own alpha.
- **The AppImage is GTK 3.** A GTK4-only check leaves the shipped Linux
  package opaque (RGBA visual still has to be proven there).

### Acceptance

- The frameless window is translucent on GTK4 with `setTransparent()`, and
  the AppImage GTK 3 build is checked the same way.
- Windows uses Acrylic with the frameless window.
- No full-window `backdrop-filter` or fading layer is introduced.
- `bun run gate` stays clean.

## Linux has no frosted-glass backdrop

Even with a translucent window, Linux has no blur behind Mortar. GNOME 51
exposes `ext_background_effect_manager_v1`; without a bind in the Wails fork
beside `setTransparent()`, the toplevel never gets a blur region.

The decided shape: in the fork beside `setTransparent()`, bind Wayland
`ext_background_effect_manager_v1` and set the toplevel surface's blur region
to the whole window, updated on resize, only while the translucent window is
on. The AppImage's GTK 3 build reaches the surface through
`gdk_wayland_window_get_wl_surface`. Needs GNOME 51. Rejected: applying blur
when the window is not translucent, or a compositor-specific non-ext protocol.

### Read

- [docs/design.md](docs/design.md#later) — the protocol and the GTK 3
  `gdk_wayland_window_get_wl_surface` path.
- The fork's `setTransparent()` work this bind sits beside.

### Traps

- **Blur only while the translucent window is on.** Leaving the region set
  after transparency is turned off blurs a solid window.
- **Resize must update the region.** A stale whole-window region from the
  previous size clips or misses the new toplevel.
- **GNOME 51 is the floor.** Older compositors without
  `ext_background_effect_manager_v1` must keep the existing solid/translucent
  behaviour, not fail to start.

### Acceptance

- On GNOME 51+, a translucent Mortar window has a whole-window blur region
  that tracks resize.
- Turning translucency off clears the blur. Compositors without the protocol
  still start.
- The AppImage GTK 3 build uses `gdk_wayland_window_get_wl_surface`.
- `bun run gate` stays clean.

## Bethesda and Minecraft families have no loaders

Mortar's catalog and loaders cover the games already in `mortar games`.
Bethesda titles need plugin load order, LOOT, script extenders and BSAs;
Minecraft needs Java instances plus Fabric, Forge and NeoForge. Neither
family can be added as a catalog-only row.

The decided shape is those loaders and game families, taken up when asked.
Rejected: pretending a folder-game catalog row is enough for plugin-order or
Java-instance games.

### Read

- [docs/design.md](docs/design.md#later) — the parked family list.
- [docs/architecture.md](docs/architecture.md#games-and-the-catalog) — when a
  game needs a driver versus a catalog row.
- `internal/settings/registry.go` — a new loader honours every existing
  setting that applies and adds a setting only for behaviour no existing one
  covers.

### Traps

- **A game normally needs no code.** These families are the exception; do not
  invent a parallel settings tree or a loader-specific launch flag.
- **GUI work ranks above CLI parity.** End users are GUI-only.

### Acceptance

- Bethesda games install with plugin load order, LOOT, script extenders and
  BSA handling; Minecraft installs as Java instances with Fabric, Forge and
  NeoForge.
- Every existing setting that applies is honoured; the registry test still
  matches `settings.Defaults()`.
- `bun run gate` stays clean.

## Several launch and import paths have never been run on real installs

Unit and sandbox coverage cannot see a real GOG install, Flatpak Steam, a
long-lived Vortex folder, OneDrive's data-folder timings, or Stardew launched
through Steam on Linux so SMAPI loads the profile's mods. Those paths stay
unmeasured until an outside tester runs them.

The decided shape is those runs, when a tester with that install is
available. Rejected: treating sandbox fixtures as proof of GOG, Flatpak
Steam, Vortex or OneDrive.

### Read

- [docs/design.md](docs/design.md#later) — the parked tester list.
- [docs/testing.md](docs/testing.md) — copied game folders, never a real
  install as the target of a write; Flatpak Steam is the exception this
  entry is for.
- [docs/games.md](docs/games.md#smapi) — Stardew's Steam launch method.

### Traps

- **Do not write through the tester's live install to "make it easier".**
  Copied folders still apply except where the run _is_ the real GOG / Flatpak
  Steam / Vortex / OneDrive path.
- **Store behaviour the maintainers cannot test follows r2modman or Gale,
  cited by file** — a tester run is evidence; it does not rewrite those
  citations without a cite of what changed.

### Acceptance

- A launch from a real GOG install and from Flatpak Steam is recorded.
- A Linux launch with Stardew's launch method set to Steam confirms SMAPI
  loads the profile's mods.
- Import from Vortex against a long-lived Vortex folder is recorded.
- Timings with the data folder under OneDrive are recorded.
- `bun run gate` stays clean. No production default changes without the
  maintainer's copied settings showing the old path.

## Mod descriptions stay in their original language

Browse and listing copy is shown as the source wrote it. Players who do not
read that language have to leave Mortar (StarModsManager already translates
Stardew descriptions). The app's own translations are a separate Lingui
catalog; mixing the two would grow the bundle with strings Mortar does not
own.

The decided shape is an optional per-description translation into the
player's language, revisited together with the app's own translations.
Rejected: translating every description unconditionally, or baking source
descriptions into the Lingui catalogs.

### Read

- [docs/design.md](docs/design.md#later) — the StarModsManager precedent and
  the "revisit together" rule.
- Frontend Lingui load path — locale catalogs load per locale so the bundle
  does not grow with each language.

### Traps

- **Do not put source-site copy in Lingui catalogs.** Those catalogs are
  Mortar's UI; per-description translation is optional and live.
- **Revisit with the app's own translations.** Shipping this first as a
  one-off source-site toggle forks the locale story.

### Acceptance

- A player can translate a mod's description into their language, optional
  and per description.
- Unsigned-out saved listings still show, marked as saved.
- `bun run gate` stays clean. Lingui catalogs do not contain source-site
  description strings.

## Share links have no hosted short-code service

Share links already name their game, but there is no hosted service for short
codes or share versioning. Codes stay local, so a share cannot be versioned
or retrieved by a short code from another machine except over LAN.

The decided shape is a hosted share service with short codes and share
versioning. Rejected: falling back to an unencrypted transfer, or a service
that re-hosts mod files (LAN profile sends between the user's own computers
remain the exception that carries files).

### Read

- [docs/architecture.md](docs/architecture.md#sharing) — the current share
  link form (game in the name).
- [docs/design.md](docs/design.md#later) — hosted short codes and versioning.

### Traps

- **Mortar never re-hosts mod files.** The hosted service is codes and
  versioning, not a CDN for archives. LAN sends between the user's computers
  remain the file-carrying exception.
- **LAN still requires the same protocol on both computers.** A newer Mortar
  refuses an older peer, code-only sends included.

### Acceptance

- A hosted short code resolves to a versioned share that names its game.
- Mod files are not re-hosted. LAN file-carrying sends still require matching
  protocol versions.
- `bun run gate` stays clean.

## Nexus listing tiles have no Add to Mortar button

Nexus downloads start only from Nexus's own Mod Manager Download button (or a
Premium API download the user asked for in Mortar). Listing tiles have no
Add to Mortar control, and adding one needs Nexus's approval first. The
browser extension already reads and marks Nexus pages and relays those
clicks; it never starts downloads or opens Nexus pages itself.

The decided shape is an Add to Mortar button on Nexus listing tiles, after
Nexus approval ([docs/nexus-application.md](docs/nexus-application.md)).
Rejected: the extension starting downloads or opening Nexus pages; shipping
the button before approval.

### Read

- [docs/nexus-application.md](docs/nexus-application.md) — the approval this
  button waits on.
- [docs/design.md](docs/design.md#later) — the tile button.
- Rethunk-Tech/mortar-browser-extension — the extension that relays Nexus
  clicks and must not start downloads.

### Traps

- **The extension never starts downloads or opens Nexus pages.** Relaying the
  official button is the only download start besides a Premium API download
  the user asked for in Mortar.
- **Nexus approval first.** A tile button without it is a terms break, not a
  feature.

### Acceptance

- Nexus listing tiles show Add to Mortar after Nexus has approved it.
- Downloads still start only from Nexus's Mod Manager Download button or a
  Premium API download the user asked for in Mortar.
- `bun run gate` stays clean. The extension still does not start downloads or
  open Nexus pages.

## Windows builds are unsigned

Windows installers and binaries ship without Authenticode. SmartScreen and
enterprise policy treat them as untrusted, which is the whole cost of not
signing.

The decided shape is Windows code signing with Azure Trusted Signing.
Rejected: a one-off purchased EV cert outside Azure Trusted Signing, or
shipping a partial sign that still trips SmartScreen.

### Read

- [docs/design.md](docs/design.md#later) — Azure Trusted Signing.
- [docs/release.md](docs/release.md) — packaging and the Windows build this
  sign attaches to.

### Traps

- **Signing is a maintainer/account decision.** This entry does not create
  the Azure Trusted Signing account; it attaches signing once that exists.
- **Do not check a signing secret into the repo or the public workflow logs.**

### Acceptance

- Windows installers and binaries produced by the release path are signed
  with Azure Trusted Signing.
- `bun run gate` stays clean. No signing material is committed.

## GitHub topic-search fallback lives in Mortar instead of go-github

`internal/source/github/fallback.go` is Mortar-specific topic-search fallback.
It cannot move until a second consumer needs it; extracting it earlier would
be a Mortar-only library split.

The decided shape is to move that fallback into go-github once a second
consumer needs it. Rejected: splitting it into go-github (or a `mortar-*`
module) for one consumer; the 200-line rule still applies, and a client for
one external service may stand alone under 200 only when it is usage-agnostic.

### Read

- `internal/source/github/fallback.go` — the fallback this move picks up.
- [AGENTS.md](AGENTS.md) — library-split rules (`go-<name>`, 200-line
  threshold, `replace ... => ../<lib>` until Mortar's gate is green).

### Traps

- **No library repo until switching Mortar to it deletes more than 200 lines
  from Mortar** (or it is a usage-agnostic client). A split only Mortar could
  use is `mortar-<name>`, which this fallback should not become.
- **Build the library change and Mortar's use together** with `replace`; tag
  the library only after Mortar's gate passes against it.

### Acceptance

- A second consumer uses the fallback from go-github, and Mortar's
  `internal/source/github/fallback.go` is gone in favour of that module.
- `bun run gate` stays clean. The replace is swapped for the tag in the
  commit that lands the feature.

## There is no macOS build

Mortar does not ship darwin/arm64 or amd64. The Linux-testable parts (Steam
roots, markers, loader branches) are under the earlier ready-to-build entry;
packaging, codesign, notarisation, Homebrew and a real Mac run are parked
until the maintainer owns or borrows a Mac. Lethal Company and most BepInEx
games ship no macOS build, so the first targets are native-macOS games and,
for Windows-only games, a CrossOver or Whisky prefix.

The decided shape is full macOS support as specified in
[docs/design.md](docs/design.md#macos-support). Catalog `darwin` path
templates and the macOS asset picker already exist; go-safefs already has a
darwin `Clone` (`clonefile`). Rejected: shipping unsigned (Gatekeeper blocks
it), adding billable macOS CI runners, or enabling MelonLoader on macOS (no
upstream macOS build; the driver already refuses Linux).

### Read

- [docs/design.md](docs/design.md#macos-support) — scope, Wails, discovery,
  loaders, packaging, testing and done-when.
- [docs/architecture.md](docs/architecture.md#stack) — the pinned Wails fork
  `wails3 build GOOS=darwin` runs on.
- go-gamelocator's Unix Steam locator (`//go:build !windows`) — today lists
  only Linux roots; add `~/Library/Application Support/Steam` (r2modman's
  darwin game-directory resolver) and `GOOS=darwin go vet ./...` in that
  module's crosscompile gate; read `steamapps/libraryfolders.vdf` as Linux
  does.
- `internal/game/install.go`, `internal/gamestore/standalone.go` — `marker`
  needs `darwinMarker` like `LinuxMarker` (`Contents/MacOS/<exe>` or
  `Contents/Resources/Data`).
- `internal/loader/bepinex5` `driver.go` (and thunderpack's) — places the
  Windows `winhttp.dll` pair; needs a darwin branch beside `contributeLinux`
  (`run_bepinex` as Steam launch option `"<profile>/run_bepinex" %command%`).
- `internal/loader/smapi/install.go` — hardcodes Linux names; the bundle is
  `Stardew Valley.app/Contents/MacOS/StardewValley`.
- `internal/nxm` — runtime `nxm://` registration that `Info.plist`
  `CFBundleURLTypes` replaces on darwin.
- `internal/loader/thunderpack/thunderpack_test.go` — the `Platform: "darwin"`
  unit-test pattern for pure-Go parts on Linux.
- `Taskfile.yml` — add `build/darwin/` (Taskfile, `Info.plist`, `icons.icns`)
  and `darwin: ./build/darwin/Taskfile.yml` under `includes`, as
  `build/windows/` and `build/linux/` do.
- r2modman darwin Steam runner (`steam_osx -applaunch <id>`) and `macos_proxy`
  launch wrapper — Gale has no macOS code.

### Traps

- **`wails3 build GOOS=darwin` is Mac-only** (cgo and the macOS SDK). Check
  the fork's WKWebView paths first: frameless title bar, `BackgroundTypeSolid`,
  drag regions, the tray (`tray_other.go`), `nxm://`, single-instance.
- **Steam strips dyld injection for hardened binaries.** Test a hardened game
  early; Doorstop's `libdoorstop.dylib` is the BepInEx path.
- **No macOS runners in workflows** (billable minutes). CI stays Linux and
  Windows; manual testing needs a lender, or a one-off VM or hosted runner
  the maintainer approves.
- **Do not ship unsigned.** Gatekeeper blocks an unnotarised app. Codesign
  (hardened runtime) and notarise (`notarytool`) need an Apple Developer ID
  (paid account, a maintainer decision).
- **MelonLoader stays off on macOS.**

### Acceptance

- `wails3 build GOOS=darwin` produces a signed, notarised `.dmg` that launches
  on macOS 14+ (arm64 and Intel) with the frameless window, tray and `nxm://`
  links.
- A Stardew Valley profile installs SMAPI, launches through Steam and loads a
  mod (`SMAPI-latest.txt`).
- Steam root and `.app` game folders are found without a manual folder choice.
- The Homebrew cask `Casks/mortar.rb` in Rethunk-Tech/homebrew-tap installs
  and updates it.
- Self-update through `wails3 updater manifest` with a `darwin` artifact in
  `Taskfile.yml`.
- `bun run gate` passes on Linux with the new darwin unit tests.
