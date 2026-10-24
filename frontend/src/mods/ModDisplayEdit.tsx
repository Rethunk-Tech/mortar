import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Link,
  TextField,
} from '@mui/material'
import { useState } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { SetEntryDisplay } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { space } from '../theme/density.ts'
import { toastError } from '../toasts/report.ts'
import { entryOf } from './lookup.ts'
import { rowLink } from './paper.ts'
import { useMods } from './store.ts'
import { openTarget } from './storeView.ts'

const MAX_DISPLAY = 100

// Changes the name, author and version this profile shows for a mod. The mod's own files and manifest stay as they
// are; clearing a field goes back to what the mod says.
export function ModDisplayEdit({ profile, mod }: { profile: Profile; mod: Mod }) {
  const { t } = useLingui()
  const entry = entryOf(profile, mod.key)
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [author, setAuthor] = useState('')
  const [version, setVersion] = useState('')
  if (!entry || (entry.mods ?? []).length !== 1) {
    return null
  }
  const begin = () => {
    setName(mod.name)
    setAuthor(mod.author)
    setVersion(mod.version)
    setOpen(true)
  }
  // A blank field, or one equal to the mod's own value, stores no override.
  const save = (values: [string, string, string]) => {
    const target = openTarget()
    if (!target) {
      return
    }
    SetEntryDisplay(target.game, target.id, mod.key, ...values)
      .then((next) => {
        useProfiles.getState().replace(next)
        return useMods.getState().load()
      })
      .then(() => setOpen(false))
      .catch((error: unknown) => toastError(t`Could not change the details`, error))
  }
  const stock = entry.mods?.[0]
  const field = (label: string, value: string, set: (v: string) => void) => (
    <TextField
      size="small"
      label={label}
      value={value}
      onChange={(e) => set(e.target.value)}
      slotProps={{ htmlInput: { maxLength: MAX_DISPLAY } }}
    />
  )
  const shownDiffers = (shown: string, stockValue: string | undefined) =>
    shown === stockValue ? '' : shown
  return (
    <>
      <Link component="button" onClick={begin} sx={rowLink}>
        {t`Edit name, author and version…`}
      </Link>
      <Dialog open={open} onClose={() => setOpen(false)} fullWidth={true} maxWidth="xs">
        <DialogTitle>{t`Edit ${{ name: mod.name }}`}</DialogTitle>
        <DialogContent dividers={true}>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap, pt: 0.5 }}>
            {field(t`Name`, name, setName)}
            {field(t`Author`, author, setAuthor)}
            {field(t`Version`, version, setVersion)}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => save(['', '', ''])}>{t`Use the mod's own`}</Button>
          <Button onClick={() => setOpen(false)}>{t`Cancel`}</Button>
          <Button
            variant="contained"
            onClick={() =>
              save([
                shownDiffers(name.trim(), stock?.name),
                shownDiffers(author.trim(), stock?.author),
                shownDiffers(version.trim(), stock?.version),
              ])
            }
          >
            {t`Save`}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  )
}
