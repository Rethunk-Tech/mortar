# Why Mortar? (vs Vortex)
<!-- tile: Nexus Mods' manager, with deep Bethesda tooling. -->

Vortex is Nexus Mods' manager, with deep Bethesda tooling. Mortar works with Nexus too and reads Vortex profiles. Use Mortar for native Linux and the Steam Deck, several sites, or to skip the account.

| | Mortar | Vortex |
| --- | --- | --- |
| Platforms | !win Windows, Linux, Steam Deck | Windows ([VORTEX_RELEASE assets](https://github.com/Nexus-Mods/Vortex/releases/latest)); SteamOS support announced ([GamingOnLinux](https://www.gamingonlinux.com/2026/01/nexus-mods-say-they-are-committed-to-supporting-vortex-on-steamos-linux)) |
| Games | GAME_COUNT | Over VORTEX_COUNT ([README](https://github.com/Nexus-Mods/Vortex)) |
| Mod sites | !win :nexus: :thunderstore: :curseforge: :github: :vsmoddb: | :nexus: ([README](https://github.com/Nexus-Mods/Vortex)) |
| Account | !win None needed; Nexus sign-in only for Nexus downloads | Nexus login to download from Nexus |
| Sharing | !win A link or a `.mortar` file; opens Nexus Collections and drafts one from a profile | Nexus Collections ([docs](https://github.com/Nexus-Mods/Vortex/blob/master/docs/mod-management/collections.md)) |
| Usage analytics | !win None | Mixpanel, keyed to the Nexus user id ([docs](https://github.com/Nexus-Mods/Vortex/blob/master/docs/analytics.md)) |
| Licence | AGPL-3.0 | GPL-3.0 |

## Where Vortex is stronger

- Deep Bethesda tooling: LOOT sorting, plugin management, file conflict views and a choice of deployment methods ([README](https://github.com/Nexus-Mods/Vortex)).
- First-party Nexus integration: Collections, endorsements and Premium download speeds in one app.

## Moving from Vortex

Profiles → **Import…**: Mortar lists Vortex profiles for the game, with a preview. Mods Vortex linked into the game folder are copied, not moved, so Vortex keeps working. A Nexus Collection link opens in Mortar like a shared profile.
