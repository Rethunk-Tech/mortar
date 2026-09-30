# AGENTS.md

Mortar is a desktop mod manager, built for several games but supporting only Stardew Valley in v1: game discovery, mod loader install, per-profile mod sets, launch, and profiles shared as links. [docs/architecture.md](docs/architecture.md) holds how it works (storage, profile semantics, Stardew and Nexus facts); [docs/design.md](docs/design.md) holds decided work not yet built, pruned as it lands; screen layout and styling rules: @docs/gui-design.md; Lethal Company research: [docs/lethal-company.md](docs/lethal-company.md).

## Milestones

Work follows [docs/design.md](docs/design.md) § Build order. Agents continue into the next milestone until the next visual deliverable, self-test in the Wails dev server's browser view (never launching the game), and hand NOMAD batched test lists.

## Decided

- Wails v3, pinned in `go.mod` (the Rethunk-AI fork) and upgraded on purpose. Pin: [docs/architecture.md](docs/architecture.md#stack). React, TypeScript and MUI on Vite; never Next.js.
- Solid window with a wallpaper backdrop under a tint, frameless with a themed title bar, and an MUI dark theme over it. Surfaces: [docs/gui-design.md](docs/gui-design.md#surfaces-and-colour). Translucent window: [docs/design.md](docs/design.md#later).
- One Go `Game` interface holds everything that differs per game: install discovery, loader install, mod source, manifest identity, and profile launch. Everything else is shared. Stardew Valley is the only implementation in v1; Lethal Company is deferred ([docs/design.md](docs/design.md#later)).
- Share links name their game. Form: [docs/architecture.md](docs/architecture.md#sharing).
- Mortar never re-hosts mod files; downloads come from each mod's own source.

## Verify

`bun run gate` is the offline gate (steps: [HUMANS.md](HUMANS.md) § Gate). Pushes to `main` spend no CI minutes: CI runs on `v*` release tags or manual dispatch only.
