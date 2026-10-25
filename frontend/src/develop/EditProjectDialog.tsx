import { useLingui } from '@lingui/react/macro'
import { Alert } from '@mui/material'
import { Hammer } from 'lucide-react'
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
import { ToneIcon } from '../shell/dialog/AppDialog.tsx'
import {
  DerivedValue,
  FieldRow,
  FormDialog,
  FormSection,
  TextInput,
} from '../shell/dialog/FormDialog.tsx'
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
  const thunderstore = sourceLabel('thunderstore')
  return (
    <FormDialog
      open={true}
      onClose={onClose}
      title={t`Edit ${{ name: info.name }}`}
      subtitle={t`Saved into the project's own files.`}
      icon={
        <ToneIcon tone="neutral">
          <Hammer size={22} />
        </ToneIcon>
      }
      submitLabel={t`Save`}
      dirty={dirty}
      busy={saving}
      onSubmit={dirty ? save : onClose}
    >
      <FormSection
        title={t`Identity`}
        description={t`Renaming moves the project's folder, files and namespace.`}
      >
        {renames && info.released ? (
          <Alert severity="warning">
            {t`Players already have ${{ version: info.released }}: a new name or id makes their game see a different mod, with fresh settings.`}
          </Alert>
        ) : null}
        <FieldRow>
          <TextInput label={t`Mod name`} value={edit.name} onChange={(name) => set({ name })} />
          <TextInput
            label={t`Id prefix`}
            value={edit.idPrefix}
            onChange={(idPrefix) => set({ idPrefix })}
          />
        </FieldRow>
        <DerivedValue
          label={t`Plugin id`}
          value={`${edit.idPrefix}.${edit.name}`}
          hint={t`Other mods know it by this.`}
        />
      </FormSection>
      <FormSection
        title={t`Listing`}
        description={t`What players see on ${{ source: thunderstore }}.`}
      >
        <TextInput
          label={t`Description`}
          value={edit.description}
          onChange={(description) => set({ description })}
          maxLength={MAX_DESCRIPTION}
          multiline={true}
        />
        <ChoicePicker
          label={t`Categories`}
          choices={categories}
          value={edit.categories ?? []}
          onChange={(next) => set({ categories: next })}
        />
      </FormSection>
      <FormSection title={t`Credits`}>
        <FieldRow>
          <TextInput label={t`Author`} value={edit.author} onChange={(author) => set({ author })} />
          <TextInput
            label={t`Team on ${{ source: thunderstore }}`}
            value={edit.team}
            onChange={(team) => set({ team })}
          />
        </FieldRow>
      </FormSection>
    </FormDialog>
  )
}
