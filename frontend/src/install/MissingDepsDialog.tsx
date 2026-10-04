import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
} from '@mui/material'
import { useEffect } from 'react'
import { useMods } from '../mods/store.ts'
import { useLocked } from '../mods/useLocked.ts'
import { download } from '../queue/actions.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
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
    <Dialog open={true} onClose={dismissOffer}>
      <DialogTitle>{t`${offer.dependentName} needs ${andList(names)}`}</DialogTitle>
      {wants.length === 0 ? (
        <DialogContent>
          <DialogContentText>
            {t`No download source is known for ${andList(names)}.`}
          </DialogContentText>
        </DialogContent>
      ) : null}
      <DialogActions>
        <Button onClick={dismissOffer}>{t`Not now`}</Button>
        {wants.length > 0 ? (
          <DisabledReason title={t`Stop the game to change mods.`} disabled={locked}>
            <Button
              variant="contained"
              disabled={locked}
              onClick={() => {
                download(wants).catch(reportUnexpected)
                dismissOffer()
              }}
            >
              {t`Add them`}
            </Button>
          </DisabledReason>
        ) : null}
      </DialogActions>
    </Dialog>
  )
}
