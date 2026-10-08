# AGENTS.md

Mortar is a desktop mod manager for several games, with Stardew Valley, Lethal Company and Valheim enabled: game discovery, mod loader install, per-profile mod sets, launch, and profiles shared as links. [docs/architecture.md](docs/architecture.md) holds how it works (storage, profile semantics, Stardew and Nexus facts); [docs/design.md](docs/design.md) holds decided work not yet built, pruned as it lands; screen layout and styling rules: @docs/gui-design.md; Lethal Company research: [docs/lethal-company.md](docs/lethal-company.md); Valheim research: [docs/valheim.md](docs/valheim.md); threat model and the guard and test for each untrusted input: [docs/security.md](docs/security.md).

## Testing

- Agents self-test everything they can: in the Wails dev server's browser view, and with the real game when a test needs it, always against a copied game folder (never symlinks into a real install: the SMAPI installer writes through them) and stopped by its recorded PID.
- Browser self-tests run against `scripts/selftest.sh` (`wails3 task selftest ARGS="start [--copy-data]"`, then `restart` after changes): a server-mode build at http://127.0.0.1:9455 with a sandboxed home, a minimal Steam library and a copied game, so the real data, game and Steam config are never touched.
- A sandbox of your own (`MORTAR_SELFTEST_DIR=/var/tmp/<name>`, 2 to 6 GB) ends with `scripts/selftest.sh destroy` in the same environment when the task is done; `stop` leaves the folder.
- `scripts/selftest.sh reap` lists marked sandboxes (regress failures included) that nothing runs from and that have been idle for 6 hours, and `--yes` deletes them.

`wails3 task selftest ARGS=seed` fills the running sandbox once, through the sandbox's own CLI, with two profiles (one from a template), mods from generated zips, history events, a manual and a scheduled backup, a stray game-Mods folder, a dot-hidden mod, an extra-mods folder, a failed download and the Config page's sources (a Stardew `config.json` changed from its shipped default, a GMCM capture with a pending edit and a result, a Lethal Company plugin's `.cfg` changed from its default, and an unowned `BepInEx.cfg`). The maintainer gets only the tests that need a human: a store login, owned games, hardware or a real desktop session.

Windows bugs: the Windows 11 KVM VM in `/var/tmp/win11-vm/` (`README.txt`, `/var/tmp/win11-vm/start.sh`, `vm.py`, a `clean-install` snapshot) is for quick repros and one-fix smokes only; long soaks are a human's.

## Decided

- Wails v3, pinned in `go.mod` (the Rethunk-AI fork) and upgraded on purpose. Pin: [docs/architecture.md](docs/architecture.md#stack). React, TypeScript and MUI on Vite; never Next.js.
- Solid window with a wallpaper backdrop under a tint, frameless with a themed title bar, and an MUI dark theme over it. Surfaces: [docs/gui-design.md](docs/gui-design.md#surfaces-and-colour). Translucent window: [docs/design.md](docs/design.md#later).
- Games, stores, runtimes, sources, metadata providers and hosts are registries joined by the signed catalog; no code pairs one with another, and a game normally needs no code. How: [docs/architecture.md](docs/architecture.md#games-and-the-catalog), adding one: [HUMANS.md](HUMANS.md#adding-a-game). Loaders are drivers in `internal/loader` (SMAPI: `internal/loader/smapi`); `internal/game`'s `Game` holds only identity and the install folder.
- Greenfield: no migration, compatibility or transition code. The maintainers' own data is migrated by hand.
- The merged all-sources browse is the primary way to get mods. Lethal Company is modded from Thunderstore, Nexus and GitHub ([docs/lethal-company.md](docs/lethal-company.md)).
- Docs never state a setting's default value. Defaults live in code (`settings.Defaults()`, read by the registry and checked by its test) and `mortar settings get` shows them; a default written in prose drifts.
- Anything tied to a game's loader, launcher, saves or content format (SMAPI options, launch method, backup/update-before-Play, run logs, Content Patcher display, default nxm profile, pre-Play check) is a per-game setting with per-profile override; Mortar-wide settings are app chrome only (theme, density, dates, notifications, data folder, LAN, Mortar's own updates).
- Stardrop and Vortex are never suggested, compared against, or contributed to (reading their source for file formats is fine).
- AUR packages are prepared and checked locally (PKGBUILD, `.SRCINFO`, makepkg/namcap in an Arch container); pushing to aur.archlinux.org needs the AUR account holder's permission for each publish, relayed by the maintainer.
- The CLI exists for agents and automation; end users are GUI-only, so GUI work ranks above CLI parity.
- Share links name their game. Form: [docs/architecture.md](docs/architecture.md#sharing).
- Mortar never re-hosts mod files; downloads come from each mod's own source. One exception: a profile sent between the user's own computers over the local network carries its mod files, from every source, when the two Mortars are paired with a code (copying between one's own machines is not distribution; pairing is Mortar's own and needs no store account); any other receiver gets the profile and downloads each mod from its source.
- Nexus downloads start only from Nexus's own Mod Manager Download button (or a Premium API download the user asked for in Mortar); the browser extension reads and marks Nexus pages and relays those clicks, never starts downloads or opens Nexus pages itself.
- The browser extension lives in its own repo, Rethunk-Tech/mortar-browser-extension, with its own version; the native-messaging protocol is versioned in the handshake (`internal/nativehost` `Protocol`, `MinProtocol`, `MaxProtocol`).

## Verify

All tests together (Go, frontend, e2e) stay under 10s warm: no duplicate coverage across layers; e2e only for flows no unit test can cover. A cold run takes about 55s and is not held to a budget (measured cost: [docs/testing.md](docs/testing.md)).

Bindings: `bun run bindings`; Lingui catalogs: `bun run --cwd frontend i18n:extract && bun run --cwd frontend i18n:compile`. Taskfile tasks run as `wails3 task <name>`; there is no standalone `task` binary. `bun run gate` is the offline gate (steps: [HUMANS.md](HUMANS.md) § Gate). CI (`.github/workflows/ci.yml`) runs the offline gate on pull requests and pushes to `main`, skipping Dependabot PRs and changes that touch only Markdown, `docs/` or issue templates, so gate locally first and batch pushes; packaging runs only on `v*` tags or manual dispatch (`release.yml`).
