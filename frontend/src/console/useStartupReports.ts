import { useLingui } from '@lingui/react/macro'
import { useCallback, useEffect } from 'react'
import type { StartupReport } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import {
  CancelMeasureNextLaunch,
  MeasureNextLaunch,
  MeasureNextLaunchPending,
  StartupReports,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { useLaunch } from '../launch/store.ts'
import { useLoaded } from '../shell/useLoaded.ts'
import { reportError, reportUnexpected } from '../toasts/report.ts'

/** The profile's startup reports, reloaded whenever a launch ends (the bridge writes one at the title screen). */
function useStartupReports(game: string, profileId: string) {
  const { t } = useLingui()
  const known = profileId !== ''
  const { data: reports, reload: reloadReports } = useLoaded<StartupReport[] | null>(
    known ? () => StartupReports(game, profileId).then((r) => r ?? []) : null,
    [game, profileId],
    null,
    reportUnexpected,
  )
  const {
    data: pending,
    setData: setPending,
    reload: reloadPending,
  } = useLoaded(
    known ? () => MeasureNextLaunchPending(game, profileId) : null,
    [game, profileId],
    false,
    reportUnexpected,
  )
  const load = useCallback(() => {
    reloadReports()
    reloadPending()
  }, [reloadReports, reloadPending])
  // A run's state changes when it starts and when it ends, after the bridge has written its report.
  useEffect(
    () =>
      useLaunch.subscribe((s, prev) => {
        if (s.status?.state !== prev.status?.state) {
          load()
        }
      }),
    [load],
  )
  const measureNext = (): void => {
    MeasureNextLaunch(game, profileId)
      .then(() => setPending(true))
      .catch(reportError(t`Could not ask for a measured launch`, measureNext))
  }
  const cancelMeasure = (): void => {
    CancelMeasureNextLaunch(game, profileId)
      .then(() => setPending(false))
      .catch(reportError(t`Could not cancel the measured launch`, cancelMeasure))
  }
  return { reports, pending, measureNext, cancelMeasure, reload: load }
}

export { useStartupReports }
