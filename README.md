<h1 align="center">Mortar</h1>

<div align="center">

![Licence](https://img.shields.io/badge/licence-AGPL--3.0-blue) ![Stack](https://img.shields.io/badge/stack-Go%20%7C%20Wails%20v3%20%7C%20React%20%7C%20MUI-blue)

</div>

---

Mortar is a desktop mod manager for more than one game. It finds your game, installs its mod loader, keeps each set of mods in its own profile and launches the game with the profile you pick. A profile is shared as a link: whoever opens it gets the same mods, with anything missing installed or queued.

The games Mortar supports come from its catalog: Stardew Valley, Lethal Company, Valheim, Vintage Story, The Sims 4 and many Thunderstore games, with mods from Nexus Mods, GitHub, Thunderstore, CurseForge and the Vintage Story Mod DB. Mortar succeeds [Concrete](https://github.com/LethalModding/Concrete).

<p align="center"><img src="build/linux/screenshots/mods.png" alt="Mortar showing a Stardew Valley profile of 805 mods with cover art" width="900"></p>

## Getting started

- Download from [mortar.rethunk.tech/download](https://mortar.rethunk.tech/download/), which has the steps for each channel: signed apt, dnf, pacman and Flatpak repositories, a [Scoop bucket](https://github.com/Rethunk-Tech/scoop-bucket) and a [Homebrew tap](https://github.com/Rethunk-Tech/homebrew-tap).
- Or take a file from [Releases](https://github.com/Rethunk-Tech/mortar/releases/latest): a Windows installer (x64 or ARM64), or for Linux an AppImage, portable program, Flatpak, `.deb`, `.rpm` or Arch package.
- The [user guide](docs/user-guide.md) covers requirements, SmartScreen on the unsigned Windows builds, first run, Nexus sign-in, profiles, backups and troubleshooting.

The optional browser extension marks Nexus Mods pages with what a profile already has and hands Mod Manager Download clicks to Mortar. It lives in [Rethunk-Tech/mortar-browser-extension](https://github.com/Rethunk-Tech/mortar-browser-extension); the user guide has its install steps.

To build it yourself:

```sh
bun install && wails3 dev
```

Prerequisites (including the `wails3` CLI built from the pinned Wails fork), build and gate: [HUMANS.md](HUMANS.md).

## Features

- Discovers the games in its catalog on Steam (including Flatpak Steam), GOG, Epic Games, the EA app, the Xbox app, Heroic, Minigalaxy, Lutris and Bottles, and installs each game's mod loader.
- Keeps each set of mods in its own profile, with install, update, rollback and share as a link or `.mortar` file.
- Downloads from Nexus Mods, Thunderstore, CurseForge, GitHub and the Vintage Story Mod DB for the games that list them, and records files saved from Patreon posts and itch.io pages; never re-hosts mod files.
- Edits a mod's config as a typed form, can keep a profile's saves separate, and shares profiles between your computers through a sync folder.
- A command line for the running app: `mortar games`, `mortar mods stardew "My Farm"`, `mortar conflicts ...`, with `--json` for scripts.
- Installs the [Mortar SMAPI Bridge](https://github.com/Rethunk-Tech/mortar-smapi-bridge) into each profile: console commands from Mortar, Generic Mod Config Menu settings, and a stream overlay. Games on BepInEx get the [Mortar BepInEx Bridge](https://github.com/Rethunk-Tech/mortar-bepinex-bridge) instead, which lets Mortar ask the running game which plugins it has loaded.

## Screenshots

<table>
<tr><td align="center"><img src="build/linux/screenshots/problems.png" alt="Problems tab explaining which mod wins each content conflict, with Disable and Make a pack win buttons" width="440"><br><sub>Problems: content conflicts explained, with one-click fixes</sub></td><td align="center"><img src="build/linux/screenshots/performance.png" alt="Performance tab: time to the title screen and each mod's startup cost" width="440"><br><sub>Performance: what each mod costs at startup</sub></td></tr>
<tr><td align="center" colspan="2"><img src="build/linux/screenshots/settings.png" alt="General settings: the antivirus that scans every mod before it is installed" width="600"><br><sub>Every mod scanned by your antivirus before it is installed</sub></td></tr>
</table>

## Documentation

| Topic | Location |
| --- | --- |
| Using Mortar | [docs/user-guide.md](docs/user-guide.md) |
| How it works | [docs/architecture.md](docs/architecture.md) |
| Decided work not yet built | [docs/design.md](docs/design.md) |
| Screens and styling | [docs/gui-design.md](docs/gui-design.md) |
| Threat model | [docs/security.md](docs/security.md) |
| Run, build, gate | [HUMANS.md](HUMANS.md) |
| Contributing | [CONTRIBUTING.md](CONTRIBUTING.md) |
| Security policy | [SECURITY.md](SECURITY.md) |
| Rules for agents | [AGENTS.md](AGENTS.md) |
| Licence | [LICENSE](LICENSE) |

## Licence

Licensed under the [GNU Affero General Public License v3.0](LICENSE).

Not affiliated with any game developer or mod site.
