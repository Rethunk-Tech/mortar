import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  MenuItem,
  TextField,
} from '@mui/material'
import { type SyntheticEvent, useEffect, useState } from 'react'
import {
  Duplicate,
  Rename,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import type { Template } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/templates/models.ts'
import { NewProfileFromTemplate } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/templates/service.ts'
import { bundleWants } from '../bundles/missingWants.ts'
import { modsLabel } from '../i18n/counts.ts'
import { useProfiles } from '../profiles/store.ts'
import { download } from '../queue/actions.ts'
import { ManageTemplatesDialog } from '../templates/TemplateDialogs.tsx'
import { useTemplates } from '../templates/useTemplates.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'

const EMPTY = ''
// A template name cannot start with a control character, so this never collides with one.
const COPY = '\u0001copy'
const MAX_NAME = 60

function StartFromSelect({
  copyOf,
  names,
  value,
  disabled,
  onChange,
  onManage,
}: {
  /** Name of the open profile, or '' when none is open. */
  copyOf: string
  names: string[]
  value: string
  disabled: boolean
  onChange: (value: string) => void
  onManage: () => void
}) {
  const { t } = useLingui()
  return (
    <>
      <TextField
        select={true}
        fullWidth={true}
        label={t`Start from`}
        value={value}
        disabled={disabled}
        onChange={(e) => onChange(e.target.value)}
        slotProps={{ select: { displayEmpty: true }, inputLabel: { shrink: true } }}
        sx={{ mt: 2 }}
      >
        <MenuItem value={EMPTY}>{t`Empty profile`}</MenuItem>
        {copyOf === '' ? null : <MenuItem value={COPY}>{t`Copy of ${{ name: copyOf }}`}</MenuItem>}
        {names.map((name) => (
          <MenuItem key={name} value={name}>
            {name}
          </MenuItem>
        ))}
      </TextField>
      <Button size="small" onClick={onManage} disabled={disabled} sx={{ mt: 0.5 }}>
        {t`Manage templates…`}
      </Button>
    </>
  )
}

function useCreate(game: string, onClose: () => void) {
  const { t } = useLingui()
  const create = useProfiles((s) => s.create)
  const [busy, run] = usePending()
  const submit = (name: string, template: Template | undefined, copyOf: string) => {
    run(
      async () => {
        if (copyOf !== '') {
          const copy = await Duplicate(game, copyOf)
          await Rename(game, copy.id, name)
          await useProfiles.getState().refresh()
          useProfiles.getState().open(copy.id)
          onClose()
          return
        }
        if (!template) {
          await create(name)
          onClose()
          return
        }
        const result = await NewProfileFromTemplate(game, template.name, name)
        await useProfiles.getState().refresh()
        useProfiles.getState().open(result.profile.id)
        onClose()
        const missing = result.missing ?? []
        const wants = bundleWants(result.missingMods)
        const queued = wants.length > 0 && (await download(wants))
        const mods = modsLabel(missing.length)
        let title = t`Created ${name}`
        if (missing.length > 0) {
          title = queued
            ? t`Created ${name}; downloading ${mods}`
            : t`Created ${name}; ${mods} still to download`
        }
        useToasts.getState().push({
          kind: missing.length > 0 && !queued ? 'warning' : 'success',
          title,
        })
      },
      { errorTitle: t`Could not create the profile` },
    )
  }
  return { busy, submit }
}

export function NewProfileDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const { templates, reload } = useTemplates(game, open)
  const { busy, submit } = useCreate(game, onClose)
  const [name, setName] = useState('')
  const [from, setFrom] = useState(EMPTY)
  const [managing, setManaging] = useState(false)
  useEffect(() => {
    if (open) {
      setName('')
      setFrom(EMPTY)
    }
  }, [open])
  const openProfile = useProfiles((s) => s.profiles.find((p) => p.id === s.openId))
  const template = templates.find((candidate) => candidate.name === from)
  const copying = from === COPY && openProfile !== undefined
  const trimmed = name.trim()
  const onSubmit = (e: SyntheticEvent) => {
    e.preventDefault()
    if (!(busy || trimmed === '')) {
      submit(trimmed, template, copying ? (openProfile?.id ?? '') : '')
    }
  }
  return (
    <>
      <Dialog open={open} onClose={busy ? undefined : onClose}>
        <form onSubmit={onSubmit}>
          <DialogTitle>{t`New profile`}</DialogTitle>
          <DialogContent>
            <TextField
              autoFocus={true}
              fullWidth={true}
              label={t`Profile name`}
              value={name}
              onChange={(e) => setName(e.target.value)}
              slotProps={{ htmlInput: { maxLength: MAX_NAME } }}
              disabled={busy}
            />
            {templates.length > 0 || openProfile !== undefined ? (
              <StartFromSelect
                copyOf={openProfile?.name ?? ''}
                names={templates.map((candidate) => candidate.name)}
                value={template || copying ? from : EMPTY}
                disabled={busy}
                onChange={setFrom}
                onManage={() => setManaging(true)}
              />
            ) : null}
          </DialogContent>
          <DialogActions sx={{ flexWrap: 'wrap' }}>
            <Button onClick={onClose} disabled={busy}>
              {t`Cancel`}
            </Button>
            <Button type="submit" variant="contained" disabled={busy || trimmed === ''}>
              {t`Create`}
            </Button>
          </DialogActions>
        </form>
      </Dialog>
      <ManageTemplatesDialog
        open={managing}
        game={game}
        templates={templates}
        onChanged={reload}
        onClose={() => setManaging(false)}
      />
    </>
  )
}
