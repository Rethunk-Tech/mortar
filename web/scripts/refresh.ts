// Re-reads the outside facts the site quotes: other managers' counts and releases, each Steam game's art addresses and
// each game's most downloaded mods.
// The release's site publish runs it so a published figure is never older than the publish.
import { refreshSteamArt } from '../src/lib/art.ts'
import { refreshRivalFacts } from '../src/lib/faq.ts'
import { playableGames, steamAppId } from '../src/lib/games.ts'
import { refreshTopMods } from '../src/lib/topmods.ts'

await refreshRivalFacts()
await refreshSteamArt(playableGames().flatMap((g) => steamAppId(g) ?? []))
await refreshTopMods(playableGames())
