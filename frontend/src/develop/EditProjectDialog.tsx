import { useLingui } from '@lingui/react/macro'
import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  TextField,
} from '@mui/material'
import { useEffect, useState } from 'react'
import type {
  Choice,
  ProjectEdit,
  ProjectInfo,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  Edit,
  TemplateForm,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { useDiscardGuard } from '../shell/useDiscardGuard.tsx'
import { space } from '../theme/density.ts'
import { usePending } from '../toasts/usePending.ts'
import { buildProject, useProjectInfo } from './builds.ts'
import { ChoicePicker } from './ChoicePicker.tsx'

const MAX_DESCRIPTION = 250

// The template's category choices; a project's own categories stay pickable even when the template lacks them.
function useCategoryChoices(game: string, current: string[]): Choice[] {
  const [choices, setChoices] = useState<Choice[]>([])
  useEffect(() => {
    TemplateForm(game)
      .then((f) =>
        setChoices((f.parameters ?? []).find((p) => p.id === 'categories')?.choices ?? []),
      )
      .catch(() => setChoices([]))
  }, [game])
  const known = new Set(choices.map((c) => c.value))
  return [...choices, ...current.filter((c) => !known.has(c)).map((c) => ({ value: c, label: c }))]
}

const editOf = (info: ProjectInfo): ProjectEdit => ({
  name: info.name,
  description: info.description,
  author: info.author,
  team: info.team,
  idPrefix: info.idPrefix,
  categories: info.categories ?? [],
})

/** Edits the details the project's files hold; a new name renames the project's folder, files and namespace. */
export function EditProjectDialog({
  game,
  dir,
  info,
  onClose,
}: {
  game: string
  dir: string
  info: ProjectInfo
  onClose: () => void
}) {
  const { t } = useLingui()
  const [edit, setEdit] = useState<ProjectEdit>(() => editOf(info))
  const [saving, run] = usePending()
  const categories = useCategoryChoices(game, edit.categories ?? [])
  const dirty = JSON.stringify(edit) !== JSON.stringify(editOf(info))
  const guard = useDiscardGuard(dirty, onClose)
  const set = (next: Partial<ProjectEdit>) => setEdit((e) => ({ ...e, ...next }))
  const renames = edit.name !== info.name || edit.idPrefix !== info.idPrefix
  const save = () =>
    run(
      async () => {
        const next = await Edit(game, dir, edit)
        useProjectInfo.setState((s) => ({ byDir: { ...s.byDir, [dir]: next } }))
        onClose()
        // A new name builds to a new output folder; building it installs the renamed mod in place of the old one.
        if (next.name !== info.name) {
          await buildProject(game, dir)
        }
      },
      { errorTitle: t`Could not save the mod's details` },
    )
  return (
    <Dialog open={true} onClose={guard.request} maxWidth="sm" fullWidth={true}>
      <form
        onSubmit={(e) => {
          e.preventDefault()
          if (!saving && dirty) {
            save()
          }
        }}
      >
        <DialogTitle>{t`Edit ${{ name: info.name }}`}</DialogTitle>
        <DialogContent
          sx={{ display: 'flex', flexDirection: 'column', gap: space.pad, pt: space.gap }}
        >
          {renames && info.released ? (
            <Alert severity="warning">
              {t`Players already have ${{ version: info.released }}: a new name or id makes their game see a different mod, with fresh settings.`}
            </Alert>
          ) : null}
          <TextField
            label={t`Mod name`}
            value={edit.name}
            onChange={(e) => set({ name: e.target.value })}
            fullWidth={true}
            helperText={t`Renames the project's folder, files and namespace.`}
          />
          <TextField
            label={t`Id prefix`}
            value={edit.idPrefix}
            onChange={(e) => set({ idPrefix: e.target.value })}
            fullWidth={true}
            helperText={t`The plugin's id: ${{ name: `${edit.idPrefix}.${edit.name}` }}`}
          />
          <TextField
            label={t`Description`}
            value={edit.description}
            onChange={(e) => set({ description: e.target.value })}
            fullWidth={true}
            multiline={true}
            minRows={2}
            helperText={`${edit.description.length}/${MAX_DESCRIPTION}`}
          />
          <ChoicePicker
            label={t`Categories on ${{ source: sourceLabel('thunderstore') }}`}
            choices={categories}
            value={edit.categories ?? []}
            onChange={(next) => set({ categories: next })}
          />
          <TextField
            label={t`Author`}
            value={edit.author}
            onChange={(e) => set({ author: e.target.value })}
            fullWidth={true}
          />
          <TextField
            label={t`Team on ${{ source: sourceLabel('thunderstore') }}`}
            value={edit.team}
            onChange={(e) => set({ team: e.target.value })}
            fullWidth={true}
          />
        </DialogContent>
        <DialogActions>
          <Button type="button" onClick={guard.request}>
            {t`Cancel`}
          </Button>
          <Button type="submit" variant="contained" disabled={saving || !dirty}>
            {t`Save`}
          </Button>
        </DialogActions>
      </form>
      {guard.dialog}
    </Dialog>
  )
}
