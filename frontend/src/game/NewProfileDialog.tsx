import { plural } from '@lingui/core/macro'
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
import type { Template } from '../../bindings/github.com/Rethunk-AI/mortar/internal/templates/models.ts'
import { NewProfileFromTemplate } from '../../bindings/github.com/Rethunk-AI/mortar/internal/templates/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { download } from '../queue/actions.ts'
import { ManageTemplatesDialog } from '../templates/TemplateDialogs.tsx'
import { templateWants } from '../templates/templateWants.ts'
import { useTemplates } from '../templates/useTemplates.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'

const EMPTY = ''
const MAX_NAME = 60

function StartFromSelect({
  names,
  value,
  disabled,
  onChange,
  onManage,
}: {
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
  const submit = (name: string, template: Template | undefined) => {
    run(
      async () => {
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
        const wants = templateWants(template, missing)
        const queued = wants.length > 0 && (await download(wants))
        const mods = plural(missing.length, { one: '# mod', other: '# mods' })
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
  const template = templates.find((candidate) => candidate.name === from)
  const trimmed = name.trim()
  const onSubmit = (e: SyntheticEvent) => {
    e.preventDefault()
    if (!(busy || trimmed === '')) {
      submit(trimmed, template)
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
            {templates.length > 0 ? (
              <StartFromSelect
                names={templates.map((candidate) => candidate.name)}
                value={template ? from : EMPTY}
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
