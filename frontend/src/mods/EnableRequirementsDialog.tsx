import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Button, Typography } from '@mui/material'
import { Power } from 'lucide-react'
import { SetModsEnabled } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { listNames } from '../i18n/list.ts'
import { useProfiles } from '../profiles/store.ts'
import { AppDialog, DialogFooter, ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { reportError } from '../toasts/report.ts'
import { localId } from './dependents.ts'
import { useEnableAsk } from './enableAsk.ts'
import { LockedReason } from './LockedReason.tsx'
import { useMods } from './store.ts'
import { announceAlso, openTarget } from './storeView.ts'
import { useLocked } from './useLocked.ts'

// Offers to switch on what a mod just switched on needs, laid out like DisableDialog.
export function EnableRequirementsDialog() {
  const { t } = useLingui()
  const offer = useEnableAsk((s) => s.offers[0])
  const dismiss = useEnableAsk((s) => s.dismiss)
  const locked = useLocked()
  if (!offer) {
    return null
  }
  const names = offer.mods.map((m) => ((m.name ?? '').trim() === '' ? localId(m.id) : m.name))
  const enableThem = () => {
    const target = openTarget()
    dismiss()
    if (!target) {
      return
    }
    SetModsEnabled(
      target.game,
      target.id,
      offer.mods.map((m) => ({ key: m.key, id: m.id })),
      true,
    )
      .then((r) => {
        useProfiles.getState().replace(r.profile)
        announceAlso(r.alsoEnabled?.length ? r.alsoEnabled : names)
        return useMods.getState().load()
      })
      .catch(reportError(t`Could not enable required mods`))
  }
  return (
    <AppDialog
      open={true}
      onClose={dismiss}
      title={t`Enable what ${{ name: offer.dependentName }} needs?`}
      subtitle={plural(
        { count: offer.mods.length },
        { one: 'It does not work while it is off.', other: 'It does not work while they are off.' },
      )}
      icon={
        <ToneIcon tone="neutral">
          <Power size={22} />
        </ToneIcon>
      }
      footer={
        <DialogFooter>
          <Button onClick={dismiss}>{t`Just this mod`}</Button>
          <LockedReason locked={locked}>
            <Button variant="contained" disabled={locked} onClick={enableThem}>
              {plural({ count: names.length }, { one: 'Enable # mod', other: 'Enable # mods' })}
            </Button>
          </LockedReason>
        </DialogFooter>
      }
    >
      <Typography>
        {t`${plural({ count: names.length }, { one: 'Also enable # mod it needs', other: 'Also enable # mods it needs' })}: ${{ names: listNames(names) }}`}
      </Typography>
    </AppDialog>
  )
}
