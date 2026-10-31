import { useLingui } from '@lingui/react/macro'
import { Alert } from '@mui/material'
import { sameValue } from '@rethunk/react-load-hooks'
import { Hammer } from 'lucide-react'
import { useEffect, useId, useState } from 'react'
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
import { DialogPanel, DialogTabs } from '../shell/dialog/DialogTabs.tsx'
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
import { ProjectDependencies } from './ProjectDependencies.tsx'
import { saveDependencies, useDependencyRows } from './projectDependencies.ts'
import { SiteIds } from './SiteIds.tsx'
import { TestWith } from './TestWith.tsx'
import { useUploadSites } from './uploadSites.ts'

const MAX_DESCRIPTION = 250

type EditTab = 'details' | 'mods' | 'sites'

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

// Thunderstore needs no ids, so Sites shows only when another site takes uploads.
function useEditTabs(sites: string[]): { id: EditTab; label: string }[] {
  const { t } = useLingui()
  return [
    { id: 'details', label: t`Details` },
    { id: 'mods', label: t`Mods` },
    ...(sites.some((s) => s !== 'thunderstore') ? [{ id: 'sites' as const, label: t`Sites` }] : []),
  ]
}

const editOf = (info: ProjectInfo): ProjectEdit => ({
  name: info.name,
  description: info.description,
  author: info.author,
  team: info.team,
  idPrefix: info.idPrefix,
  categories: info.categories ?? [],
  testWith: info.testWith ?? [],
  nexus: info.nexus,
  curseforge: info.curseforge,
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
  const sites = useUploadSites(game, dir)
  const categories = useCategoryChoices(game, edit.categories ?? [])
  const details = !sameValue(edit, editOf(info))
  const deps = useDependencyRows(game, dir)
  const dirty = details || deps.dirty
  const set = (next: Partial<ProjectEdit>) => setEdit((e) => ({ ...e, ...next }))
  // A mod of files has no project file to rename or build: only its listing, credits and dependencies change.
  const files = Boolean(info.kind)
  const renames = !files && (edit.name !== info.name || edit.idPrefix !== info.idPrefix)
  const save = () =>
    run(
      async () => {
        // Dependencies first: a rename moves the project's files.
        if (deps.dirty) {
          await saveDependencies(game, dir, deps.loaded, deps.rows)
        }
        const next = details ? await Edit(game, dir, edit) : info
        useProjectInfo.setState((s) => ({ byDir: { ...s.byDir, [dir]: next } }))
        onClose()
        // A new name or id builds to a new assembly, installing the renamed mod in place of the old one; changed
        // dependencies are referenced and declared on the plugin as it builds.
        if (!files && (renames || deps.dirty)) {
          await buildProject(game, dir)
        }
      },
      { errorTitle: t`Could not save the mod's details` },
    )
  const thunderstore = sourceLabel('thunderstore')
  const ids = useId()
  const [tab, setTab] = useState<EditTab>('details')
  const tabs = useEditTabs(sites)
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
      issue={deps.loading ? t`Reading the dependencies…` : ''}
      dirty={dirty}
      busy={saving}
      onSubmit={dirty ? save : onClose}
      tabs={<DialogTabs ids={ids} tab={tab} onTab={setTab} tabs={tabs} label={t`Mod details`} />}
    >
      <DialogPanel ids={ids} panel="details" open={tab === 'details'}>
        {files ? null : (
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
        )}
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
            {files ? null : (
              <TextInput
                label={t`Author`}
                value={edit.author}
                onChange={(author) => set({ author })}
              />
            )}
            <TextInput
              label={t`Team on ${{ source: thunderstore }}`}
              value={edit.team}
              onChange={(team) => set({ team })}
            />
          </FieldRow>
        </FormSection>
      </DialogPanel>
      <DialogPanel ids={ids} panel="mods" open={tab === 'mods'}>
        <ProjectDependencies
          game={game}
          rows={deps.rows}
          setRows={deps.setRows}
          required={files}
          loading={deps.loading}
          error={deps.error}
          onRetry={deps.retry}
        />
        <TestWith
          game={game}
          ids={edit.testWith ?? []}
          onChange={(testWith) => set({ testWith })}
        />
      </DialogPanel>
      <DialogPanel ids={ids} panel="sites" open={tab === 'sites'}>
        <SiteIds
          sites={sites}
          nexus={edit.nexus}
          curseforge={edit.curseforge}
          onNexus={(nexus) => set({ nexus })}
          onCurseForge={(curseforge) => set({ curseforge })}
        />
      </DialogPanel>
    </FormDialog>
  )
}
