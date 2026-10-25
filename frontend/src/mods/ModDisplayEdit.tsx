import { useLingui } from '@lingui/react/macro'
import { Button, Link } from '@mui/material'
import { Pencil } from 'lucide-react'
import { useState } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { SetEntryDisplay } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { FormDialog, TextInput } from '../shell/dialog/FormDialog.tsx'
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
    <TextInput label={label} value={value} onChange={set} maxLength={MAX_DISPLAY} />
  )
  const shownDiffers = (shown: string, stockValue: string | undefined) =>
    shown === stockValue ? '' : shown
  return (
    <>
      <Link component="button" onClick={begin} sx={rowLink}>
        {t`Edit name, author and version…`}
      </Link>
      <FormDialog
        open={open}
        onClose={() => setOpen(false)}
        size="sm"
        title={t`Edit ${{ name: mod.name }}`}
        icon={
          <ToneIcon tone="neutral">
            <Pencil size={22} />
          </ToneIcon>
        }
        submitLabel={t`Save`}
        dirty={name !== mod.name || author !== mod.author || version !== mod.version}
        onSubmit={() =>
          save([
            shownDiffers(name.trim(), stock?.name),
            shownDiffers(author.trim(), stock?.author),
            shownDiffers(version.trim(), stock?.version),
          ])
        }
      >
        {field(t`Name`, name, setName)}
        {field(t`Author`, author, setAuthor)}
        {field(t`Version`, version, setVersion)}
        <Button onClick={() => save(['', '', ''])} sx={{ alignSelf: 'flex-start' }}>
          {t`Use the mod's own`}
        </Button>
      </FormDialog>
    </>
  )
}
