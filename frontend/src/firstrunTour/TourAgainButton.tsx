import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { SetTipsSeen } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useSettings } from '../settings/store.ts'
import { errorText } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useTourReplay } from './replay.ts'
import { tourClearSeen } from './seen.ts'

function TourAgainButton() {
  const { t } = useLingui()
  return (
    <Button
      variant="outlined"
      onClick={() => {
        const seen = useSettings.getState().tipsSeen
        SetTipsSeen(tourClearSeen(seen)).catch((err: unknown) => {
          const body = errorText(err)
          useToasts.getState().push({
            kind: 'error',
            title: t`Couldn't save that setting`,
            ...(body ? { body } : {}),
          })
        })
        useTourReplay.getState().request()
      }}
    >
      {t`Show the tour again`}
    </Button>
  )
}

export { TourAgainButton }
