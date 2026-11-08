import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { SetTipsSeen } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { settingSaveFailed } from '../settings/persist.ts'
import { useSettings } from '../settings/store.ts'
import { useTourReplay } from './replay.ts'
import { tourClearSeen } from './seen.ts'

function TourAgainButton() {
  const { t } = useLingui()
  return (
    <Button
      variant="outlined"
      onClick={() => {
        const seen = useSettings.getState().tipsSeen
        SetTipsSeen(tourClearSeen(seen)).catch(settingSaveFailed)
        useTourReplay.getState().request()
      }}
    >
      {t`Show the tour again`}
    </Button>
  )
}

export { TourAgainButton }
