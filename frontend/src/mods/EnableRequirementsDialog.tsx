import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  List,
  ListItem,
} from '@mui/material'
import { SetModsEnabled } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { andList } from '../install/missingDeps.ts'
import { useProfiles } from '../profiles/store.ts'
import { useEnableAsk } from './enableAsk.ts'
import { paper } from './paper.ts'
import { useMods } from './store.ts'
import { announceAlso, fail, open } from './storeView.ts'

export function EnableRequirementsDialog() {
  const { t } = useLingui()
  const offer = useEnableAsk((s) => s.offers[0])
  const dismiss = useEnableAsk((s) => s.dismiss)
  if (!offer) {
    return null
  }
  const names = offer.mods.map((m) => m.name || m.uniqueId)
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
    <Dialog open={true} onClose={dismiss} slotProps={{ paper }} transitionDuration={0}>
      <DialogTitle>{t`${offer.dependentName} needs ${andList(names)}`}</DialogTitle>
      <DialogContent>
        <List dense={true}>
          {offer.mods.map((m) => (
            <ListItem key={`${m.key}:${m.uniqueId}`}>{m.name || m.uniqueId}</ListItem>
          ))}
        </List>
      </DialogContent>
      <DialogActions>
        <Button onClick={dismiss} sx={{ whiteSpace: 'nowrap' }}>
          {t`Just this mod`}
        </Button>
        <Button variant="contained" onClick={enableThem} sx={{ whiteSpace: 'nowrap' }}>
          {t`Enable them too`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
