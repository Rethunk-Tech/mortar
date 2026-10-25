import { useLingui } from '@lingui/react/macro'
import { MenuItem, TextField } from '@mui/material'
import { useEffect, useState } from 'react'
import type { Versions } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  NextVersions,
  Release,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'

/** Picks the next version, takes what changed, and records both: a CHANGELOG entry, its commit, and the tag. */
export function ReleaseDialog({
  game,
  dir,
  name,
  onClose,
}: {
  game: string
  dir: string
  name: string
  onClose: () => void
}) {
  const { t } = useLingui()
  const [versions, setVersions] = useState<Versions | null>(null)
  const [version, setVersion] = useState('')
  const [notes, setNotes] = useState('')
  const [pending, run] = usePending()
  useEffect(() => {
    NextVersions(game, dir)
      .then((v) => {
        setVersions(v)
        setVersion(v.patch)
      })
      .catch(() => setVersions(null))
  }, [game, dir])
  const choices = versions ? [...new Set([versions.patch, versions.minor, versions.major])] : []
  const lines = notes.split('\n').filter((l) => l.trim() !== '')
  const release = () =>
    run(
      async () => {
        await Release(game, dir, version, lines)
        useToasts.getState().push({
          kind: 'success',
          title: t`Released ${{ name }} ${{ version }}`,
          body: t`Push the commit and the tag to publish it.`,
        })
        onClose()
      },
      { errorTitle: t`Could not release the mod` },
    )
  return (
    <ConfirmDialog
      open={true}
      title={t`Release ${{ name }}`}
      confirmLabel={t`Release`}
      busy={pending}
      confirmDisabled={version === '' || lines.length === 0}
      fieldFocus={true}
      maxWidth={480}
      onCancel={onClose}
      onConfirm={release}
    >
      <TextField
        select={true}
        size="small"
        fullWidth={true}
        label={t`Version`}
        value={version}
        onChange={(e) => setVersion(e.target.value)}
        sx={{ mb: 2 }}
        helperText={
          versions?.current
            ? t`Last released: ${{ version: versions.current }}`
            : t`Not released yet`
        }
      >
        {choices.map((v) => (
          <MenuItem key={v} value={v}>
            {v}
          </MenuItem>
        ))}
      </TextField>
      <TextField
        multiline={true}
        minRows={3}
        fullWidth={true}
        label={t`What changed, one line each`}
        value={notes}
        onChange={(e) => setNotes(e.target.value)}
      />
    </ConfirmDialog>
  )
}
