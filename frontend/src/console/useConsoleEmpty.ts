import { Runs } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { useLaunch } from '../launch/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useLoaded } from '../shell/useLoaded.ts'
import { useShownEntries } from './logHooks.ts'
import { canSendTo, useConsole } from './store.ts'

function useHasRuns(game: string, profile: string, running: boolean): boolean {
  const { data: hasRuns } = useLoaded(
    running ? null : () => Runs(game, profile).then((list) => (list ?? []).length > 0),
    [game, profile, running],
    true,
  )
  return hasRuns
}

export function useConsoleEmpty(game: string): boolean {
  const entries = useShownEntries()
  const openId = useProfiles((s) => s.openId)
  const viewingRun = useConsole((s) => s.viewingRun)
  const running = useLaunch((s) => canSendTo(s.status, game, openId))
  const hasRuns = useHasRuns(game, openId, running)
  return entries.length === 0 && viewingRun === '' && !running && !hasRuns
}
