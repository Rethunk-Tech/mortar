# Why Mortar? (vs Stardrop)
<!-- tile: A mod manager made for Stardew Valley alone. -->

Stardrop is a Stardew Valley manager; Mortar reads its profiles. Use Mortar to search for mods in-app, have SMAPI installed, or manage more than Stardew.

| | Mortar | Stardrop |
| --- | --- | --- |
| Platforms | Windows, Linux, Steam Deck | Windows, Linux, macOS ([STARDROP_RELEASE](https://github.com/Floogen/Stardrop/releases/latest)) |
| Games | GAME_COUNT | 1, Stardew Valley ([docs](https://floogen.gitbook.io/stardrop/what-is-stardrop)) |
| SMAPI | !win Installed for you; updates offered | Must be installed first; updates offered ([docs](https://floogen.gitbook.io/stardrop/getting-started/installing-stardrop/linux)) |
| Finding mods | !win Browse searches :nexus: :curseforge: :github: | Local archives, or Mod Manager Download on :nexus: ([docs](https://floogen.gitbook.io/stardrop/guides/using-stardrop/adding-a-mod)) |
| Sharing | !win A link or a `.mortar` file | A JSON export ([source](https://github.com/Floogen/Stardrop/blob/main/Stardrop/Views/ProfileExportWindow.axaml.cs)) |
| Conflicts | !win Content Patcher conflicts explained in Problems | Duplicate mods skipped, missing dependencies reported |
| Licence | AGPL-3.0 | GPL-3.0 |

## Where Stardrop is stronger

- Native macOS builds.
- Years of tuning for one game: per-profile mod configs and Nexus Collections support ([features](https://floogen.gitbook.io/stardrop/guides/nexus-mods-integration/features)).

## Moving from Stardrop

In Profiles, choose **Import…** and Mortar lists Stardrop profiles, with a preview. Stardrop and the game's Mods folder are left as they were.
