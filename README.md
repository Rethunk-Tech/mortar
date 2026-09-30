<h1 align="center">Mortar</h1>

<div align="center">

![Licence](https://img.shields.io/badge/licence-AGPL--3.0-blue) ![Stack](https://img.shields.io/badge/stack-Go%20%7C%20Wails%20v3%20%7C%20React%20%7C%20MUI-blue)

</div>

---

Mortar is a desktop mod manager, built for more than one game. It finds your game, installs their mod loader, keeps each set of mods in its own profile, and launches the game with the profile you pick. A profile is shared as a link: whoever opens it gets the same mods, with anything missing installed or queued and every dependency checked.

Version 1 supports Stardew Valley (SMAPI, Nexus Mods) only; Lethal Company (BepInEx, Thunderstore) is deferred to a later release. Mortar succeeds [Concrete](https://github.com/LethalModding/Concrete).

## Quick start

```sh
bun install && wails3 dev
```

Prerequisites, build and gate: [HUMANS.md](HUMANS.md).

## Documentation

| Topic | Location |
| --- | --- |
| How it works now | [docs/architecture.md](docs/architecture.md) |
| Decided work not yet built | [docs/design.md](docs/design.md) |
| Screens and styling | [docs/gui-design.md](docs/gui-design.md) |
| Run, build, gate | [HUMANS.md](HUMANS.md) |
| Contributing | [CONTRIBUTING.md](CONTRIBUTING.md) |
| Security policy | [SECURITY.md](SECURITY.md) |
| Rules for agents | [AGENTS.md](AGENTS.md) |
| Licence | [LICENSE](LICENSE) |

## Licence

Licensed under the [GNU Affero General Public License v3.0](LICENSE).
