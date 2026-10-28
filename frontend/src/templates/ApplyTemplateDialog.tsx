import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { MenuItem, Typography } from '@mui/material'
import { LayoutTemplate } from 'lucide-react'
import { useEffect, useState } from 'react'
import type {
  Preview,
  Template,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/templates/models.ts'
import {
  ApplyTemplate,
  PreviewApplyTemplate,
  UndoApplyTemplate,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/templates/service.ts'
import { savedGroupWants } from '../groups/missingWants.ts'
import { listNames } from '../i18n/list.ts'
import { useProfiles } from '../profiles/store.ts'
import { download } from '../queue/actions.ts'
import { ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { FormDialog, SelectField } from '../shell/dialog/FormDialog.tsx'
import { ErrorRetry } from '../shell/ErrorRetry.tsx'
import { useLoaded } from '../shell/useLoaded.ts'
import { inlineError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'

function PreviewGroups({ preview }: { preview: Preview }) {
  const { t } = useLingui()
  const groups = [
    {
      title: plural(
        { count: preview.add?.length ?? 0 },
        { one: 'Adds # mod', other: 'Adds # mods' },
      ),
      names: preview.add ?? [],
    },
    {
      title: t`Already has ${{ count: preview.alreadyHave?.length ?? 0 }}`,
      names: preview.alreadyHave ?? [],
    },
    {
      title: t`Different version ${{ count: preview.versionDiffers?.length ?? 0 }} (kept as is)`,
      names: preview.versionDiffers ?? [],
    },
    {
      title: t`Settings changes ${{ count: preview.settingsChanges?.length ?? 0 }}`,
      names: preview.settingsChanges ?? [],
    },
  ]
  return (
    <>
      {groups.map((group) => (
        <div key={group.title}>
          <Typography sx={{ mt: 1.5, fontWeight: 600 }}>{group.title}</Typography>
          {group.names.length > 0 ? (
            <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
              {listNames(group.names)}
            </Typography>
          ) : null}
        </div>
      ))}
    </>
  )
}

function useApply(game: string, profileId: string, onClose: () => void) {
  const { t } = useLingui()
  const [busy, run] = usePending()
  const apply = (name: string) =>
    run(
      async () => {
        const result = await ApplyTemplate(game, name, profileId)
        useProfiles.getState().replace(result.profile)
        onClose()
        const missing = result.missing ?? []
        const wants = savedGroupWants(result.missingMods)
        const queued = wants.length > 0 && (await download(wants))
        const left = missing.length
        const added = plural(
          { count: result.added },
          {
            one: `Added # mod from ${name}`,
            other: `Added # mods from ${name}`,
          },
        )
        let rest = ''
        if (left > 0) {
          rest = queued
            ? plural(left, { one: 'downloading # mod', other: 'downloading # mods' })
            : plural(left, { one: '# mod still to download', other: '# mods still to download' })
        }
        const title = rest ? t`${added}; ${rest}` : added
        useToasts.getState().push({
          kind: missing.length > 0 && !queued ? 'warning' : 'success',
          title,
          action: {
            label: t`Undo`,
            profileId,
            run: async () => {
              useProfiles.getState().replace(await UndoApplyTemplate(game, profileId, result.undo))
            },
          },
        })
      },
      { errorTitle: t`Could not apply the template` },
    )
  return { busy, apply }
}

export function ApplyTemplateDialog({
  open,
  game,
  profileId,
  templates,
  onClose,
}: {
  open: boolean
  game: string
  profileId: string
  templates: Template[]
  onClose: () => void
}) {
  const { t } = useLingui()
  const [name, setName] = useState('')
  const { busy, apply } = useApply(game, profileId, onClose)
  const {
    data: preview,
    error,
    reload,
  } = useLoaded<Preview | null>(
    open && name ? () => PreviewApplyTemplate(game, name, profileId) : null,
    [open, game, name, profileId],
    null,
  )
  let issue = ''
  if (!preview && name !== '') {
    issue = error ? t`Could not preview the template` : t`Reading the template…`
  }
  useEffect(() => {
    if (open) {
      setName(templates[0]?.name ?? '')
    }
  }, [open, templates])
  return (
    <FormDialog
      open={open}
      onClose={onClose}
      size="sm"
      title={t`Apply a template`}
      icon={
        <ToneIcon tone="neutral">
          <LayoutTemplate size={22} />
        </ToneIcon>
      }
      submitLabel={t`Apply`}
      issue={issue}
      dirty={false}
      busy={busy}
      onSubmit={() => apply(name)}
    >
      <SelectField label={t`Template`} value={name} onChange={setName} disabled={busy}>
        {templates.map((template) => (
          <MenuItem key={template.name} value={template.name}>
            {template.name}
          </MenuItem>
        ))}
      </SelectField>
      {error ? <ErrorRetry error={inlineError(error)} onRetry={reload} /> : null}
      {preview ? <PreviewGroups preview={preview} /> : null}
    </FormDialog>
  )
}
