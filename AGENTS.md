# AGENTS.md

Mortar is a desktop mod manager, built for several games but supporting only Stardew Valley in v1: game discovery, mod loader install, per-profile mod sets, launch, and profiles shared as links. [docs/architecture.md](docs/architecture.md) holds how it works (storage, profile semantics, Stardew and Nexus facts); [docs/design.md](docs/design.md) holds decided work not yet built, pruned as it lands; screen layout and styling rules: @docs/gui-design.md; Lethal Company research: [docs/lethal-company.md](docs/lethal-company.md).

## Testing

Agents self-test everything they can: in the Wails dev server's browser view, and with the real game when a test needs it, always against a copied game folder (never a symlink) and stopped by its recorded PID. Browser self-tests run against `scripts/selftest.sh` (`wails3 task selftest -- start [--copy-data]`, then `restart` after changes): a server-mode build at http://127.0.0.1:9455 with a sandboxed home, a minimal Steam library and a copied game, so the real data, game and Steam config are never touched. NOMAD gets only the tests that need a human: a store login, owned games, hardware or a real desktop session.

## Decided

- Wails v3, pinned in `go.mod` (the Rethunk-AI fork) and upgraded on purpose. Pin: [docs/architecture.md](docs/architecture.md#stack). React, TypeScript and MUI on Vite; never Next.js.
- Solid window with a wallpaper backdrop under a tint, frameless with a themed title bar, and an MUI dark theme over it. Surfaces: [docs/gui-design.md](docs/gui-design.md#surfaces-and-colour). Translucent window: [docs/design.md](docs/design.md#later).
- One Go `Game` interface holds everything that differs per game: install discovery, loader install, mod source, manifest identity, and profile launch. Everything else is shared. Stardew Valley is the only implementation in v1; Lethal Company is deferred ([docs/design.md](docs/design.md#later)).
- Share links name their game. Form: [docs/architecture.md](docs/architecture.md#sharing).
- Mortar never re-hosts mod files; downloads come from each mod's own source. One exception: a profile sent over the local network carries its mod files when both Mortars are signed in to the same Nexus account (the user's own machines); any other receiver gets the profile and downloads each mod from its source.
- Nexus downloads start only from Nexus's own Mod Manager Download button (or a Premium API download the user asked for in Mortar); the browser extension reads and marks Nexus pages and relays those clicks, never starts downloads or opens Nexus pages itself.

## Verify

Bindings: `bun run bindings`; Lingui catalogs: `bun run --cwd frontend i18n:extract && bun run --cwd frontend i18n:compile`. Taskfile tasks run as `wails3 task <name>`; there is no standalone `task` binary. `bun run gate` is the offline gate (steps: [HUMANS.md](HUMANS.md) § Gate). CI (`.github/workflows/ci.yml`) runs the offline gate on pull requests and pushes to `main`, so gate locally first and batch pushes; packaging runs only on `v*` tags or manual dispatch (`release.yml`).
