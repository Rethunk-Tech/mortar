import { useLingui } from '@lingui/react/macro'
import { Alert } from '@mui/material'
import { sameValue } from '@rethunk/react-load-hooks'
import { Hammer } from 'lucide-react'
import { memo, useCallback, useEffect, useId, useMemo, useState } from 'react'
import type {
  Choice,
  ProjectEdit,
  ProjectInfo,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  Edit,
  TemplateForm,
  UploadSites,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { ThunderstoreCategories } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/service.ts'
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
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { buildProject, useProjectInfo } from './builds.ts'
import { ChoicePicker } from './ChoicePicker.tsx'
import { DependencySync } from './DependencySync.tsx'
import { ProjectDependencies } from './ProjectDependencies.tsx'
import { saveDependencies, useDependencyRows } from './projectDependencies.ts'
import { SiteIds } from './SiteIds.tsx'
import { TestWith } from './TestWith.tsx'
import { THUNDERSTORE_DESCRIPTION_MAX } from './uploadSites.ts'

const MAX_DESCRIPTION = THUNDERSTORE_DESCRIPTION_MAX

type EditTab = 'details' | 'mods' | 'sites'

async function categoryChoices(game: string): Promise<Choice[]> {
  const listed = await ThunderstoreCategories(game)
    .then((l) => (l ?? []).map((c) => ({ value: c.slug, label: c.name })))
    .catch(() => [])
  if (listed.length > 0) {
    return listed
  }
  return TemplateForm(game)
    .then((f) => (f.parameters ?? []).find((p) => p.id === 'categories')?.choices ?? [])
    .catch(() => [])
}

// The upload sites and the community's Thunderstore categories, read together: each arrival redraws the whole
// dialog. The template's choices stand in when Thunderstore's list is unavailable, and a project's own categories
// stay pickable even when neither lists them.
function useEditChoices(game: string, dir: string, current: string[]) {
  const [read, setRead] = useState<{ sites: string[]; choices: Choice[] }>({
    sites: [],
    choices: [],
  })
  useEffect(() => {
    Promise.all([
      UploadSites(game, dir)
        .then((list) => (list ?? []).map((s) => s.id))
        .catch(() => []),
      categoryChoices(game),
    ])
      .then(([sites, choices]) => setRead({ sites, choices }))
      .catch(reportUnexpected)
  }, [game, dir])
  const known = new Set(read.choices.map((c) => c.value))
  const currentKey = current.filter((c) => !known.has(c)).join('\n')
  // Stable while the read and the project's own categories are, so the panel drawing them skips a redraw when a
  // dependency read lands.
  const categories = useMemo(() => {
    const knownNow = new Set(read.choices.map((c) => c.value))
    return [
      ...read.choices,
      ...currentKey
        .split('\n')
        .filter((c) => c !== '' && !knownNow.has(c))
        .map((c) => ({ value: c, label: c })),
    ]
  }, [read.choices, currentKey])
  return { sites: read.sites, categories }
}

// Thunderstore needs no ids, so Sites shows only when another site takes uploads.
function useEditTabs(sites: string[]): { id: EditTab; label: string }[] {
  const { t } = useLingui()
  return useMemo(
    () => [
      { id: 'details', label: t`Details` },
      { id: 'mods', label: t`Mods` },
      ...(sites.some((s) => s !== 'thunderstore')
        ? [{ id: 'sites' as const, label: t`Sites` }]
        : []),
    ],
    [sites, t],
  )
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

// The Details tab's fields: their own component, so a dependency read landing after the dialog opens redraws the tab
// bar and the Mods panel and not these.
const DetailsPanel = memo(function DetailsPanelView({
  edit,
  set,
  files,
  renames,
  released,
  categories,
}: {
  edit: ProjectEdit
  set: (next: Partial<ProjectEdit>) => void
  files: boolean
  renames: boolean
  released: string
  categories: Choice[]
}) {
  const { t } = useLingui()
  const thunderstore = sourceLabel('thunderstore')
  return (
    <>
      {files ? null : (
        <FormSection
          title={t`Identity`}
          description={t`Renaming moves the project's folder, files and namespace.`}
        >
          {renames && released ? (
            <Alert severity="warning">
              {t`Players already have ${{ version: released }}: a new name or id makes their game see a different mod, with fresh settings.`}
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
    </>
  )
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
  const { sites, categories } = useEditChoices(game, dir, edit.categories ?? [])
  const details = !sameValue(edit, editOf(info))
  const deps = useDependencyRows(game, dir)
  const dirty = details || deps.dirty
  const set = useCallback((next: Partial<ProjectEdit>) => setEdit((e) => ({ ...e, ...next })), [])
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
        <DetailsPanel
          edit={edit}
          set={set}
          files={files}
          renames={renames}
          released={info.released}
          categories={categories}
        />
      </DialogPanel>
      <DialogPanel ids={ids} panel="mods" open={tab === 'mods'}>
        <DependencySync game={game} dir={dir} onChanged={deps.retry} />
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
