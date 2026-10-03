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
    <Dialog
      open={true}
      onClose={dismiss}
      slotProps={{ paper }}
      transitionDuration={0}
      maxWidth="sm"
      fullWidth={true}
    >
      <DialogTitle sx={{ whiteSpace: 'normal', overflowWrap: 'anywhere' }}>
        {t`${offer.dependentName} needs ${andList(names)}`}
      </DialogTitle>
      <DialogContent>
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
