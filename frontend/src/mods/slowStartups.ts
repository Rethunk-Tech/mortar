import { useEffect, useState } from 'react'
import { StartupReports } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { type SlowStartup, slowStartups } from '../console/startupView.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'

export function useSlowStartups(): SlowStartup[] {
  const game = useProfiles((s) => s.game?.id ?? '')
  const profileId = useProfiles((s) => s.openId)
  const [rows, setRows] = useState<SlowStartup[]>([])
  useEffect(() => {
    if (game === '' || profileId === '') {
      return
    }
    StartupReports(game, profileId)
      .then((reports) => setRows(reports?.[0] ? slowStartups(reports[0]) : []))
      .catch(reportUnexpected)
  }, [game, profileId])
  return rows
}
