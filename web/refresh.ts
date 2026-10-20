// Re-reads the outside facts the site quotes: other managers' counts and releases, and each Steam game's art addresses.
// The release's site publish runs it so a published figure is never older than the publish.
import { refreshSteamArt } from './src/lib/art.ts'
import { refreshRivalFacts } from './src/lib/faq.ts'
import { playableGames, steamAppId } from './src/lib/games.ts'

await refreshRivalFacts()
await refreshSteamArt(playableGames().flatMap((g) => steamAppId(g) ?? []))
