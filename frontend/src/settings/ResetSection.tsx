import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
} from '@mui/material'
import { useState } from 'react'
import { SetByKey } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { pushUndoToast } from '../toasts/undo.ts'
import { specByKey, usePrefSpecs } from './prefSpecs.ts'
import { GAME_STARDEW, prefAsString, prefRaw, specGameArg } from './prefValue.ts'
import { useSettings } from './store.ts'

export function ResetSectionButton({ keys }: { keys: string[] }) {
  const { t } = useLingui()
  const [open, setOpen] = useState(false)
  const specs = usePrefSpecs()
  const settings = useSettings()
  const push = useToasts((s) => s.push)
  const game = settings.lastGame || GAME_STARDEW
  const run = () => {
    const snaps = keys.flatMap((key) => {
      const spec = specByKey(specs, key)
      if (!spec) {
        return []
      }
      return [
        {
          key,
          value: prefAsString(prefRaw(settings, spec, game), spec),
          gameArg: specGameArg(spec, game),
          def: spec.default,
        },
      ]
    })
    const restore = () =>
      Promise.all(snaps.map((s) => SetByKey(s.key, s.value, s.gameArg))).catch(() => undefined)
    setOpen(false)
    Promise.all(snaps.map((s) => SetByKey(s.key, s.def, s.gameArg)))
      .then(() => {
        pushUndoToast(push, t`Section reset to defaults`, t`Undo`, restore)
      })
      .catch((e: unknown) => {
        push({ kind: 'error', title: t`Could not reset this section.`, body: errorMessage(e) })
      })
  }
  if (keys.length === 0) {
    return null
  }
  return (
    <>
      <Button variant="text" onClick={() => setOpen(true)} sx={{ alignSelf: 'flex-start' }}>
        {t`Reset section to defaults`}
      </Button>
      <Dialog open={open} onClose={() => setOpen(false)} transitionDuration={0}>
        <DialogTitle>{t`Reset this section?`}</DialogTitle>
        <DialogContent>
          <DialogContentText>
            {t`Restore every setting in this group to its default. You can undo from the toast.`}
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)}>{t`Cancel`}</Button>
          <Button color="warning" variant="contained" onClick={run}>
            {t`Reset`}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  )
}
