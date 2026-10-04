import { useLingui } from '@lingui/react/macro'
import { List, ListItem } from '@mui/material'
import { SetModsEnabled } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { andList } from '../install/missingDeps.ts'
import { useProfiles } from '../profiles/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { useEnableAsk } from './enableAsk.ts'
import { useMods } from './store.ts'
import { announceAlso, fail, open } from './storeView.ts'

export function EnableRequirementsDialog() {
  const { t } = useLingui()
  const offer = useEnableAsk((s) => s.offers[0])
  const dismiss = useEnableAsk((s) => s.dismiss)
  if (!offer) {
    return null
  }
  const names = offer.mods.map((m) => ((m.name ?? '').trim() === '' ? t`Unknown mod` : m.name))
  const enableThem = () => {
    const target = open()
    dismiss()
    if (!target) {
      return
    }
    SetModsEnabled(
      target.game,
      target.id,
      offer.mods.map((m) => ({ key: m.key, uniqueId: m.uniqueId })),
      true,
    )
      .then((r) => {
        useProfiles.getState().replace(r.profile)
        announceAlso(r.alsoEnabled?.length ? r.alsoEnabled : names)
        return useMods.getState().load()
      })
      .catch(fail(t`Could not enable required mods`))
  }
  return (
    <ConfirmDialog
      open={true}
      title={t`${offer.dependentName} needs ${andList(names)}`}
      confirmLabel={t`Switch them on too`}
      cancelLabel={t`Just this mod`}
      onCancel={dismiss}
      onConfirm={enableThem}
    >
      <List dense={true}>
        {offer.mods.map((m) => (
          <ListItem
            key={`${m.key}:${m.uniqueId}`}
            title={(m.name ?? '').trim() === '' ? m.uniqueId : undefined}
          >
            {(m.name ?? '').trim() === '' ? t`Unknown mod` : m.name}
          </ListItem>
        ))}
      </List>
    </ConfirmDialog>
  )
}
