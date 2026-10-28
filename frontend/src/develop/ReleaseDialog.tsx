import { useLingui } from '@lingui/react/macro'
import { MenuItem } from '@mui/material'
import { Tag } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { Versions } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  NextVersions,
  Release,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { FormDialog, SelectField, TextInput } from '../shell/dialog/FormDialog.tsx'
import { errorKind } from '../toasts/errorKind.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { useTrustAsk } from './builds.ts'

/** Picks the next version, takes what changed, and records both: a CHANGELOG entry, its commit, and the tag. */
export function ReleaseDialog({
  game,
  dir,
  name,
  onClose,
  onLocal,
}: {
  game: string
  dir: string
  name: string
  onClose: () => void
  // The release stayed local (no GitHub remote): Mortar offers to upload it itself.
  onLocal: (version: string) => void
}) {
  const { t } = useLingui()
  const [versions, setVersions] = useState<Versions | null>(null)
  const [version, setVersion] = useState('')
  const [notes, setNotes] = useState('')
  const [drafted, setDrafted] = useState('')
  const [pending, run] = usePending()
  useEffect(() => {
    NextVersions(game, dir)
      .then((v) => {
        setVersions(v)
        setVersion(v.patch)
        // The commits since the last release are a draft of what changed, edited before it is recorded.
        const draft = (v.changes ?? []).join('\n')
        setDrafted(draft)
        setNotes((n) => n || draft)
      })
      .catch(() => setVersions(null))
  }, [game, dir])
  const choices = versions ? [...new Set([versions.patch, versions.minor, versions.major])] : []
  const lines = notes.split('\n').filter((l) => l.trim() !== '')
  const release = () =>
    run(
      async () => {
        let remote: string
        try {
          remote = await Release(game, dir, version, lines)
        } catch (e) {
          if (errorKind(e) !== 'unconfirmed') {
            throw e
          }
          useTrustAsk.setState({ ask: { game, dir, pack: false } })
          onClose()
          return
        }
        useToasts.getState().push({
          kind: 'success',
          title: t`Released ${{ name }} ${{ version }}`,
          ...(remote
            ? {
                body: t`Pushed to ${{ remote }}; its CI publishes it to ${{ source: sourceLabel('thunderstore') }}.`,
              }
            : {}),
        })
        if (remote) {
          onClose()
        } else {
          onLocal(version)
        }
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
      subtitle={
        versions?.github
          ? t`Commits the changelog, tags the version and pushes both to ${{ remote: versions.github }}, whose CI publishes it.`
          : t`Commits the changelog and tags the version here, then offers to upload it.`
      }
      icon={
        <ToneIcon tone="neutral">
          <Tag size={22} />
        </ToneIcon>
      }
      size="sm"
      submitLabel={t`Release`}
      issue={issue}
      dirty={notes.trim() !== drafted.trim()}
      busy={pending}
      onSubmit={release}
    >
      <SelectField
        label={t`Version`}
        value={version}
        onChange={setVersion}
        hint={
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
      </SelectField>
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
