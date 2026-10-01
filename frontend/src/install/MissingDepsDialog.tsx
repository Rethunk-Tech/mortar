import { useLingui } from '@lingui/react/macro'
import { Button, Dialog, DialogActions, DialogTitle } from '@mui/material'
import { useEffect } from 'react'
import { paper } from '../mods/paper.ts'
import { useMods } from '../mods/store.ts'
import { useLocked } from '../mods/useLocked.ts'
import { download } from '../queue/actions.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { andList, depName, stillMissing, wantsOf } from './missingDeps.ts'
import { useInstall } from './store.ts'

export function MissingDepsDialog() {
  const { t } = useLingui()
  const offer = useInstall((s) => s.offers[0])
  const dismissOffer = useInstall((s) => s.dismissOffer)
  const problems = useMods((s) => s.problems)
  const locked = useLocked()
  const missing = offer ? stillMissing(offer, problems) : []
  useEffect(() => {
    if (offer && missing.length === 0) {
      dismissOffer()
    }
  }, [offer, missing.length, dismissOffer])
  if (!offer || missing.length === 0) {
    return null
  }
  const names = missing.map(depName)
  const wants = wantsOf(missing)
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
