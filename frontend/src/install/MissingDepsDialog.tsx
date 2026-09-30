import { useLingui } from '@lingui/react/macro'
import { Button, Dialog, DialogActions, DialogTitle } from '@mui/material'
import { paper } from '../mods/paper.ts'
import { useLocked } from '../mods/useLocked.ts'
import { download } from '../queue/actions.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { andList, depName, wantsOf } from './missingDeps.ts'
import { useInstall } from './store.ts'

export function MissingDepsDialog() {
  const { t } = useLingui()
  const offer = useInstall((s) => s.offers[0])
  const dismissOffer = useInstall((s) => s.dismissOffer)
  const locked = useLocked()
  if (!offer) {
    return null
  }
  const names = offer.missing.map(depName)
  const wants = wantsOf(offer.missing)
  return (
    <Dialog open={true} onClose={dismissOffer} slotProps={{ paper }} transitionDuration={0}>
      <DialogTitle>{t`${offer.dependentName} needs ${andList(names)}`}</DialogTitle>
      <DialogActions>
        <Button onClick={dismissOffer} sx={{ whiteSpace: 'nowrap' }}>
          {t`Not now`}
        </Button>
        {wants.length > 0 ? (
          <Button
            variant="contained"
            disabled={locked}
            onClick={() => {
              download(wants).catch(reportUnexpected)
              dismissOffer()
            }}
            sx={{ whiteSpace: 'nowrap' }}
          >
            {t`Add them`}
          </Button>
        ) : null}
      </DialogActions>
    </Dialog>
  )
}
