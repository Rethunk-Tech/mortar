import { useLingui } from '@lingui/react/macro'
import { MenuItem, TextField } from '@mui/material'
import { Tag } from 'lucide-react'
import { useEffect, useId, useState } from 'react'
import type { Versions } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  NextVersions,
  Release,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { Field, FormDialog, TextInput } from '../shell/dialog/FormDialog.tsx'
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
  const versionId = useId()
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
  let issue = ''
  if (version === '') {
    issue = t`Pick the version.`
  } else if (lines.length === 0) {
    issue = t`Say what changed.`
  }
  return (
    <FormDialog
      open={true}
      onClose={onClose}
      title={t`Release ${{ name }}`}
      subtitle={t`Adds the changelog entry, commits it and tags the version.`}
      icon={
        <ToneIcon tone="neutral">
          <Tag size={22} />
        </ToneIcon>
      }
      size="sm"
      submitLabel={t`Release`}
      issue={issue}
      dirty={notes.trim() !== ''}
      busy={pending}
      onSubmit={release}
    >
      <Field
        label={t`Version`}
        htmlFor={versionId}
        hint={
          versions?.current
            ? t`Last released: ${{ version: versions.current }}`
            : t`Not released yet`
        }
      >
        <TextField
          id={versionId}
          select={true}
          size="small"
          fullWidth={true}
          value={version}
          onChange={(e) => setVersion(e.target.value)}
        >
          {choices.map((v) => (
            <MenuItem key={v} value={v}>
              {v}
            </MenuItem>
          ))}
        </TextField>
      </Field>
      <TextInput
        label={t`What changed, one line each`}
        value={notes}
        onChange={setNotes}
        multiline={true}
        autoFocus={true}
      />
    </FormDialog>
  )
}
