import { StartupReports } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { type SlowStartup, slowStartups } from '../console/startupView.ts'
import { useProfiles } from '../profiles/store.ts'
import { useLoaded } from '../shell/useLoaded.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useMods } from './store.ts'

const NO_ROWS: SlowStartup[] = []

export function useSlowStartups(): SlowStartup[] {
  const game = useProfiles((s) => s.game?.id ?? '')
  const profileId = useProfiles((s) => s.openId)
  const problemsAt = useMods((s) => s.problemsAt)
  return useLoaded(
    game === '' || profileId === ''
      ? null
      : () =>
          StartupReports(game, profileId).then((reports) =>
            reports?.[0] ? slowStartups(reports[0]) : NO_ROWS,
          ),
    [game, profileId, problemsAt],
    NO_ROWS,
    reportUnexpected,
  ).data
}
