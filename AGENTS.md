# AGENTS.md

Mortar is a multi-game desktop mod manager: game discovery, mod loader install, per-profile mod sets, launch, and profiles shared as links. [docs/design.md](docs/design.md) holds decided work not yet built, pruned as it lands.

## Planning phase

No code, scaffolding, generated project files or worker dispatch until NOMAD gives an explicit go-ahead. Until then the work is research, testing decisions against evidence, and question rounds, recorded in `docs/design.md`.

## Decided

- Wails v3, pinned to one exact beta and upgraded on purpose. React, TypeScript and MUI on Vite; never Next.js.
- The look carries over from Concrete: a translucent window (Acrylic on Windows, translucent on Linux, base `rgba(25,25,30,0.8)`), frameless with a themed title bar, and an MUI dark theme whose paper and background are 80% opaque so the backdrop shows through. Only the colour tokens change.
- One Go `Game` interface holds everything that differs per game: install discovery, loader install, mod source, manifest identity, and profile launch. Everything else is shared. Stardew Valley is the first implementation, Lethal Company the second.
- Share links name their game: `mortar://<game>/...`.
- Mortar never re-hosts mod files; downloads come from each mod's own source.
