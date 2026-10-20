# Why Mortar? (vs r2modman)

r2modman is the long-standing Thunderstore manager; Mortar reads its profiles. Use Mortar for games outside Thunderstore, or for mods from more than one site.

| | Mortar | r2modman |
| --- | --- | --- |
| Platforms | Windows, Linux, Steam Deck | Windows, Linux, Steam Deck ([README](https://github.com/ebkr/r2modmanPlus)) |
| Games | GAME_COUNT, including Stardew Valley, Vintage Story and The Sims 4 | R2MODMAN_COUNT Thunderstore games ([ecosystem.json](https://github.com/ebkr/r2modmanPlus/blob/develop/src/assets/data/ecosystem.json)) |
| Mod sites | !win :nexus: :thunderstore: :curseforge: :github: :vsmoddb: | :thunderstore: ([README](https://github.com/ebkr/r2modmanPlus)) |
| Sharing | !win A link or a `.mortar` file; opens r2modman codes and `.r2z` files | A code or an `.r2z` file ([source](https://github.com/ebkr/r2modmanPlus/blob/develop/src/r2mm/mods/ProfileImportExport.ts)) |
| Import from other managers | !win r2modman, Gale, Vortex, Mod Organizer 2, Stardrop | Its own codes and files |
| Licence | AGPL-3.0 | MIT |

## Where r2modman is stronger

- More Thunderstore games: R2MODMAN_COUNT configured against Mortar's THUNDERSTORE_COUNT, fed straight from Thunderstore's own schema.
- Years of use by a large community, and its profile codes are the format other Thunderstore managers share.

## Moving from r2modman

Profiles → **Import…**: Mortar lists r2modman profiles for the game, with a preview. A profile code or `.r2z` file works through **Use a code or file…**. r2modman profiles are left as they were.
