# Why Mortar? (vs Gale)

Gale is an open-source Thunderstore manager; Mortar reads its profiles. Use Mortar for games outside Thunderstore, or for Nexus and other sites in the same profile.

| | Mortar | Gale |
| --- | --- | --- |
| Platforms | Windows, Linux, Steam Deck | Windows, Linux ([README](https://github.com/Kesomannen/gale)) |
| Games | GAME_COUNT, including Stardew Valley, Vintage Story and The Sims 4 | GALE_COUNT Thunderstore games ([games.json](https://github.com/Kesomannen/gale/blob/master/src-tauri/games.json)) |
| Mod sites | !win :nexus: :thunderstore: :curseforge: :github: :vsmoddb: | :thunderstore: :hexium: ([README](https://github.com/Kesomannen/gale)) |
| Sharing | A link or a `.mortar` file; opens Gale and r2modman codes and files | Codes and files compatible with r2modman, modpack upload to Thunderstore ([Features](https://github.com/Kesomannen/gale/wiki/Features)) |
| Import from other managers | !win r2modman, Gale, Vortex, Mod Organizer 2, Stardrop | r2modman and Thunderstore Mod Manager ([Features](https://github.com/Kesomannen/gale/wiki/Features)) |
| Licence | AGPL-3.0 | GPL-3.0 |

## Where Gale is stronger

- Translated into ten languages; Mortar is English only for now.
- Uploads a profile to Thunderstore as a modpack; Mortar exports the modpack zip for you to upload.
- A zoom setting ([preferences page](https://github.com/Kesomannen/gale/blob/master/src/routes/prefs/%2Bpage.svelte)).

## Moving from Gale

Profiles → **Import…**: Mortar lists Gale profiles for the game, with a preview. A Gale code or file works through **Use a code or file…**. Gale profiles are left as they were.
