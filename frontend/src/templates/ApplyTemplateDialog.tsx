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
  Typography,
} from '@mui/material'
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
import { bundleWants } from '../bundles/missingWants.ts'
import { useProfiles } from '../profiles/store.ts'
import { download } from '../queue/actions.ts'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'

function PreviewGroups({ preview }: { preview: Preview }) {
  const { t } = useLingui()
  const groups = [
    {
      title: plural(preview.add?.length ?? 0, { one: 'Adds # mod', other: 'Adds # mods' }),
      names: preview.add ?? [],
    },
    { title: t`Already has ${preview.alreadyHave?.length ?? 0}`, names: preview.alreadyHave ?? [] },
    {
      title: t`Different version ${preview.versionDiffers?.length ?? 0} (kept as is)`,
      names: preview.versionDiffers ?? [],
    },
    {
      title: t`Settings changes ${preview.settingsChanges?.length ?? 0}`,
      names: preview.settingsChanges ?? [],
    },
  ]
  return (
    <>
      {groups.map((group) => (
        <div key={group.title}>
          <Typography sx={{ mt: 1.5, fontWeight: 600 }}>{group.title}</Typography>
          {group.names.length > 0 ? (
            <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
              {group.names.join(', ')}
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
        const wants = bundleWants(result.missingMods)
        const queued = wants.length > 0 && (await download(wants))
        const left = missing.length
        let title = plural(result.added, {
          one: `Added # mod from ${name}`,
          other: `Added # mods from ${name}`,
        })
        if (left > 0) {
          title = queued
            ? plural(result.added, {
                one: `Added # mod from ${name}; downloading ${plural(left, { one: '# mod', other: '# mods' })}`,
                other: `Added # mods from ${name}; downloading ${plural(left, { one: '# mod', other: '# mods' })}`,
              })
            : plural(result.added, {
                one: `Added # mod from ${name}; ${plural(left, { one: '# mod', other: '# mods' })} still to download`,
                other: `Added # mods from ${name}; ${plural(left, { one: '# mod', other: '# mods' })} still to download`,
              })
        }
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
  const [preview, setPreview] = useState<Preview | null>(null)
  const { busy, apply } = useApply(game, profileId, onClose)
  useEffect(() => {
    if (open) {
      setName(templates[0]?.name ?? '')
    }
  }, [open, templates])
  useEffect(() => {
    setPreview(null)
    if (!(open && name)) {
      return
    }
    let live = true
    PreviewApplyTemplate(game, name, profileId)
      .then((found) => live && setPreview(found))
      .catch(reportError(t`Could not preview the template`))
    return () => {
      live = false
    }
  }, [open, game, name, profileId, t])
  return (
    <Dialog open={open} onClose={busy ? undefined : onClose}>
      <DialogTitle>{t`Apply a template`}</DialogTitle>
      <DialogContent sx={{ minWidth: 380, maxWidth: 'calc(100vw - 64px)' }}>
        <TextField
          select={true}
          fullWidth={true}
          label={t`Template`}
          value={name}
          disabled={busy}
          onChange={(e) => setName(e.target.value)}
          sx={{ mt: 1 }}
        >
          {templates.map((template) => (
            <MenuItem key={template.name} value={template.name}>
              {template.name}
            </MenuItem>
          ))}
        </TextField>
        {preview ? <PreviewGroups preview={preview} /> : null}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={busy}>
          {t`Cancel`}
        </Button>
        <Button variant="contained" disabled={busy || !preview} onClick={() => apply(name)}>
          {t`Apply`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
