# AGENTS.md

Mortar is a multi-game desktop mod manager: game discovery, mod loader install, per-profile mod sets, launch, and profiles shared as links. [docs/design.md](docs/design.md) holds decided work not yet built, pruned as it lands; screen layout and styling rules: @docs/gui-design.md; Lethal Company research: [docs/lethal-company.md](docs/lethal-company.md).

## Milestones

Milestone 1 (shell and look, `docs/design.md` § Build order) is approved (NOMAD, 2026-09-29). Every later milestone needs NOMAD's explicit go-ahead before any of its code is written.

## Decided

- Wails v3, pinned to one exact beta and upgraded on purpose. React, TypeScript and MUI on Vite; never Next.js.
- The look carries over from Concrete: a translucent window (Acrylic on Windows, translucent on Linux, base `rgba(25,25,30,0.8)`), frameless with a themed title bar, and an MUI dark theme whose paper and background are 80% opaque so the backdrop shows through. Only the colour tokens change.
- One Go `Game` interface holds everything that differs per game: install discovery, loader install, mod source, manifest identity, and profile launch. Everything else is shared. Stardew Valley is the first implementation, Lethal Company the second.
- Share links name their game: `https://mortar.rethunk.tech/<game>/p#<payload>`, handed to the app as `mortar://<game>/p/<payload>`.
- Mortar never re-hosts mod files; downloads come from each mod's own source.

## Verify

`bun run gate` is the offline gate: Biome, golangci-lint, `tsc --noEmit`, `go vet` and `go test`, stopping at the first failure. The pre-push hook runs it. CI runs it plus `wails3 build` only on `v*` release tags (or manual dispatch); pushes to `main` spend no CI minutes.
