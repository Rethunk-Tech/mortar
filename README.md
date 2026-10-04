<h1 align="center">Mortar</h1>

<div align="center">

![Licence](https://img.shields.io/badge/licence-AGPL--3.0-blue) ![Stack](https://img.shields.io/badge/stack-Go%20%7C%20Wails%20v3%20%7C%20React%20%7C%20MUI-blue)

</div>

---

Mortar is a desktop mod manager, built for more than one game. It finds your game, installs its mod loader, keeps each set of mods in its own profile, and launches the game with the profile you pick. A profile is shared as a link: whoever opens it gets the same mods, with anything missing installed or queued and every dependency checked.

Version 1 supports Stardew Valley (SMAPI, Nexus Mods) only; Lethal Company (BepInEx, Thunderstore) is deferred to a later release. Mortar succeeds [Concrete](https://github.com/LethalModding/Concrete).

## Getting started

Download the latest build from [Releases](https://github.com/Rethunk-AI/mortar/releases/latest) or the [download page](https://mortar.rethunk.tech/download/): a Windows installer (x64 or ARM64), or for Linux an AppImage, Flatpak, `.deb`, `.rpm` or Arch package. Mortar finds Stardew Valley from Steam, GOG, Heroic or Lutris and installs SMAPI itself. The [user guide](docs/user-guide.md) walks through installing, first run, Nexus sign-in, profiles, backups and troubleshooting. The Windows builds are not code-signed, so SmartScreen warns the first time one runs: choose **More info**, then **Run anyway**.

The optional browser extension marks Nexus Mods pages with what a profile already has and hands Mod Manager Download clicks to Mortar. Until it is listed in the browser stores, take `mortar-browser-extension.zip` from the same release, unzip it, open `chrome://extensions` in Chrome, Edge or another Chromium browser, turn on Developer mode and choose **Load unpacked** on the unzipped folder. In Firefox, open `about:debugging#/runtime/this-firefox`, choose **Load Temporary Add-on** and pick its `manifest.json`; Firefox removes a temporary add-on when it restarts.

To build it yourself:

```sh
bun install && wails3 dev
```

Prerequisites (including the `wails3` CLI built from the pinned Wails fork), build and gate: [HUMANS.md](HUMANS.md).

## Features

- Discovers Stardew Valley on Steam (including Flatpak Steam), GOG, Heroic and Lutris, and installs SMAPI.
- Keeps each set of mods in its own profile, with install, update, rollback and share as a link or `.mortar` file.
- Downloads from Nexus Mods and GitHub; never re-hosts mod files.
- A command line for the running app: `mortar games`, `mortar mods stardew "My Farm"`, `mortar conflicts ...`, with `--json` for scripts.
- Installs the [Mortar SMAPI Bridge](https://github.com/Rethunk-AI/mortar-smapi-bridge) into each profile: console commands from Mortar, Generic Mod Config Menu settings, and a stream overlay.

## Documentation

| Topic | Location |
| --- | --- |
| Using Mortar | [docs/user-guide.md](docs/user-guide.md) |
| How it works | [docs/architecture.md](docs/architecture.md) |
| Decided work not yet built | [docs/design.md](docs/design.md) |
| Screens and styling | [docs/gui-design.md](docs/gui-design.md) |
| Run, build, gate | [HUMANS.md](HUMANS.md) |
| Contributing | [CONTRIBUTING.md](CONTRIBUTING.md) |
| Security policy | [SECURITY.md](SECURITY.md) |
| Rules for agents | [AGENTS.md](AGENTS.md) |
| Licence | [LICENSE](LICENSE) |

## Licence

Licensed under the [GNU Affero General Public License v3.0](LICENSE).
