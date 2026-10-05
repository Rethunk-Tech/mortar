import { Start as StartBisect } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/bisect/service.ts'
import { Runs } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { useCommandPalette } from './store.ts'

export async function startCrashBisectFromPalette(): Promise<string | null> {
  const { game, openId } = useProfiles.getState()
  if (!(game?.id && openId)) {
    return 'Open a profile first.'
  }
  const runs = await Runs(game.id, openId)
  // A run the player stopped counts only when it crashed before the stop.
  const crashed = (runs ?? []).find(
    (run) => run.outcome === 'crashed' || (run.errors > 0 && !run.exit?.stopped),
  )
  if (!crashed) {
    return 'No crashed run found for this profile.'
  }
  if (crashed.cause !== null && crashed.cause !== undefined) {
    return 'A suspected mod is already known for the latest crash.'
  }
  const id = await StartBisect(game.id, openId)
  useCommandPalette.getState().setBisect({ id, game: game.id, profile: openId })
  return null
}
