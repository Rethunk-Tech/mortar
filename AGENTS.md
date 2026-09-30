# AGENTS.md

Mortar is a multi-game desktop mod manager: game discovery, mod loader install, per-profile mod sets, launch, and profiles shared as links. [docs/design.md](docs/design.md) holds decided work not yet built, pruned as it lands; screen layout and styling rules: @docs/gui-design.md; Lethal Company research: [docs/lethal-company.md](docs/lethal-company.md).

## Milestones

Milestone 1 (shell and look) is approved (NOMAD, 2026-09-29) and Milestone 2 (Stardew core) (NOMAD, 2026-09-30), per `docs/design.md` § Build order. On 2026-09-30 NOMAD authorized working on through later milestones until the next visual deliverable, self-testing in the Wails dev server's browser view (short of launching the game), and handing over batches with many things to test.

## Decided

- Wails v3, pinned to one exact beta and upgraded on purpose. React, TypeScript and MUI on Vite; never Next.js.
- The look carries over from Concrete: a solid window (base `rgb(25,25,30)`) with a wallpaper backdrop under a static `rgba(25,25,30,0.8)` tint, frameless with a themed title bar, and an MUI dark theme whose paper and background are 80% opaque over it. Only the colour tokens change. The Concrete-style translucent window is deferred to v2 (design.md § Later).
- One Go `Game` interface holds everything that differs per game: install discovery, loader install, mod source, manifest identity, and profile launch. Everything else is shared. Stardew Valley is the first implementation, Lethal Company the second.
- Share links name their game: `https://mortar.rethunk.tech/<game>/p#<payload>`, handed to the app as `mortar://<game>/p/<payload>`.
- Mortar never re-hosts mod files; downloads come from each mod's own source.

## Verify

`bun run gate` is the offline gate: Biome, golangci-lint, `tsc --noEmit`, `go vet` and `go test`, stopping at the first failure. The pre-push hook runs it. CI runs it plus `wails3 build` only on `v*` release tags (or manual dispatch); pushes to `main` spend no CI minutes.
