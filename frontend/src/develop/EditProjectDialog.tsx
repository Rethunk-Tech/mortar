import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Chip, TextField, Typography } from '@mui/material'
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
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { space } from '../theme/density.ts'
import { usePending } from '../toasts/usePending.ts'
import { buildProject, useProjectInfo } from './builds.ts'

// The template's category choices, for labels; a project's own categories show even when the template lacks them.
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

function Categories({
  game,
  value,
  set,
}: {
  game: string
  value: string[]
  set: (v: string[]) => void
}) {
  const { t } = useLingui()
  const choices = useCategoryChoices(game, value)
  return (
    <Box>
      <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary', mb: 0.5 }}>
        {t`Categories on ${{ source: sourceLabel('thunderstore') }}`}
      </Typography>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75 }}>
        {choices.map((c) => {
          const on = value.includes(c.value)
          return (
            <Chip
              key={c.value}
              size="small"
              label={c.label}
              color={on ? 'primary' : 'default'}
              variant={on ? 'filled' : 'outlined'}
              onClick={() => set(on ? value.filter((v) => v !== c.value) : [...value, c.value])}
            />
          )
        })}
      </Box>
    </Box>
  )
}

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
  const [edit, setEdit] = useState<ProjectEdit>({
    name: info.name,
    description: info.description,
    author: info.author,
    team: info.team,
    idPrefix: info.idPrefix,
    categories: info.categories ?? [],
  })
  const [pending, run] = usePending()
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
  const field = (key: 'name' | 'description' | 'author' | 'team' | 'idPrefix', label: string) => (
    <TextField
      size="small"
      label={label}
      value={edit[key]}
      onChange={(e) => set({ [key]: e.target.value })}
    />
  )
  return (
    <ConfirmDialog
      open={true}
      title={t`Edit ${{ name: info.name }}`}
      confirmLabel={t`Save`}
      busy={pending}
      fieldFocus={true}
      maxWidth={520}
      onCancel={onClose}
      onConfirm={save}
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.pad }}>
        {renames && info.released ? (
          <Alert variant="compact" severity="warning">
            {t`Players already have ${{ version: info.released }}: a new name or id makes their game see a different mod, with fresh settings.`}
          </Alert>
        ) : null}
        <Box sx={{ display: 'flex', gap: space.gap }}>
          {field('name', t`Mod name`)}
          {field('idPrefix', t`Id prefix`)}
        </Box>
        {field('description', t`Description`)}
        <Box sx={{ display: 'flex', gap: space.gap }}>
          {field('author', t`Author`)}
          {field('team', t`Team on ${{ source: sourceLabel('thunderstore') }}`)}
        </Box>
        <Categories
          game={game}
          value={edit.categories ?? []}
          set={(categories) => set({ categories })}
        />
      </Box>
    </ConfirmDialog>
  )
}
