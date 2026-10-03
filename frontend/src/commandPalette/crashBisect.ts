import { Start as StartBisect } from '../../bindings/github.com/Rethunk-AI/mortar/internal/bisect/service.ts'
import { Runs } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import { canBisectCrash } from '../console/canBisect.ts'
import { useProfiles } from '../profiles/store.ts'
import { useCommandPalette } from './store.ts'

export async function startCrashBisectFromPalette(): Promise<string | null> {
  const { game, openId } = useProfiles.getState()
  if (!(game?.id && openId)) {
    return 'Open a profile first.'
  }
  const runs = await Runs(game.id, openId)
  const crashed = (runs ?? []).find((run) => run.errors > 0)
  if (!crashed) {
    return 'No crashed run found for this profile.'
  }
  if (!canBisectCrash({ cause: crashed.cause })) {
    return 'A suspected mod is already known for the latest crash.'
  }
  const id = await StartBisect(game.id, openId)
  useCommandPalette.getState().setBisect({ id, game: game.id, profile: openId })
  return null
}
