import { useLingui } from '@lingui/react/macro'
import { Button, Typography } from '@mui/material'
import { useEffect } from 'react'
import { listNames } from '../i18n/list.ts'
import { LockedReason } from '../mods/LockedReason.tsx'
import { useMods } from '../mods/store.ts'
import { useLocked } from '../mods/useLocked.ts'
import { download } from '../queue/actions.ts'
import { AppDialog, DialogFooter } from '../shell/dialog/AppDialog.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { depName, stillMissing, wantsOf } from './missingDeps.ts'
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
    <AppDialog
      open={true}
      onClose={dismissOffer}
      size="sm"
      title={t`${{ name: offer.dependentName }} needs ${{ names: listNames(names) }}`}
      footer={
        <DialogFooter>
          <Button onClick={dismissOffer}>{t`Not now`}</Button>
          {wants.length > 0 ? (
            <LockedReason locked={locked}>
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
            </LockedReason>
          ) : null}
        </DialogFooter>
      }
    >
      {wants.length === 0 ? (
        <Typography sx={{ color: 'text.secondary' }}>
          {t`No download source is known for ${{ names: listNames(names, names.length) }}.`}
        </Typography>
      ) : null}
    </AppDialog>
  )
}
