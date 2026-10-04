import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { SetTipsSeen } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useSettings } from '../settings/store.ts'
import { reportError } from '../toasts/report.ts'
import { useTourReplay } from './replay.ts'
import { tourClearSeen } from './seen.ts'

function TourAgainButton() {
  const { t } = useLingui()
  return (
    <Button
      variant="outlined"
      onClick={() => {
        const seen = useSettings.getState().tipsSeen
        SetTipsSeen(tourClearSeen(seen)).catch(reportError(t`Could not save that setting`))
        useTourReplay.getState().request()
      }}
    >
      {t`Show the tour again`}
    </Button>
  )
}

export { TourAgainButton }
