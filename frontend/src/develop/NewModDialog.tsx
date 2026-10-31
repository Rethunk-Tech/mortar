import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button, FormControlLabel, MenuItem, Switch, TextField } from '@mui/material'
import { readStored, writeStored } from '@rethunk/react-load-hooks'
import { Hammer } from 'lucide-react'
import { memo, useCallback, useEffect, useId, useState } from 'react'
import type {
  Form,
  GitHubOwner,
  Tools,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  Create,
  Tools as ReadTools,
  TemplateForm,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { PickFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { useGameInfo } from '../games/info.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { Field, FieldRow, FormDialog, SelectField, TextInput } from '../shell/dialog/FormDialog.tsx'
import { space } from '../theme/density.ts'
import { reportError, toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { CONTENT_KIND, useWizardForm } from './assetKinds.ts'
import { buildProject } from './builds.ts'
import { DependenciesStep } from './DependenciesStep.tsx'
import { startDependencies, useKnownPackages } from './dependencies.ts'
import {
  chosenDependencies,
  type Draft,
  draftIssue,
  firstDraft,
  isRemembered,
  REMEMBERED,
  rememberedOf,
  type Step,
  stepIndex,
  stepIssue,
  stepParams,
  validModName,
  wizardSteps,
} from './draft.ts'
import { LicenseChart } from './LicenseChart.tsx'
import { Params, type SetDraft } from './Params.tsx'
import { RepositoryChoice } from './RepositoryChoice.tsx'
import { sdkSatisfies } from './sdk.ts'

// The profile picker's value for a new development profile; Create takes it as no profile.
const NEW_PROFILE = 'new'
const LICENCES = [
  ['apache-2.0', 'Apache 2.0'],
  ['gpl-3.0', 'GNU GPL 3.0'],
  ['mpl-2.0', 'Mozilla Public License 2.0'],
  ['unlicense', 'The Unlicense'],
] as const

const NO_OWNERS: GitHubOwner[] = []

// One element for every render, so the header's icon is not redrawn with each step.
const ICON = (
  <ToneIcon tone="neutral">
    <Hammer size={22} />
  </ToneIcon>
)

// Memoised: stepping through the wizard leaves it alone.
const Prerequisites = memo(function PrerequisitesView({
  tools,
  form,
}: {
  tools: Tools | null
  form: Form | null
}) {
  const { t } = useLingui()
  if (!tools) {
    return null
  }
  const sdks = tools.dotnetSdks ?? []
  const need = form?.sdk
  let sdk = ''
  if (sdks.length === 0) {
    sdk = t`Install the .NET SDK to build mods.`
  } else if (need?.version && !sdkSatisfies(need, sdks)) {
    sdk = t`This template builds with the .NET SDK ${{ version: need.version }}; install it to build mods.`
  }
  const missing = [
    sdk,
    tools.git ? '' : t`Install git to keep your mod's history.`,
    tools.gameDir
      ? ''
      : t`Mortar cannot find the game, so the mod builds against its published libraries.`,
  ].filter((line) => line !== '')
  return missing.length === 0 ? null : (
    <Alert variant="compact" severity="warning">
      {missing.join(' ')}
    </Alert>
  )
})

// Memoised on what it shows, so the GitHub owner filled in as the tools arrive leaves the fields alone.
const ModStep = memo(
  function ModStepView({
    draft,
    form,
    noCodeOffered,
    set,
  }: {
    draft: Draft
    form: Form | null
    noCodeOffered: boolean
    set: SetDraft
  }) {
    const { t } = useLingui()
    return (
      <>
        <FieldRow>
          <TextInput
            label={t`Mod name`}
            error={
              draft.name === '' || validModName(draft.name)
                ? ''
                : t`Letters and digits, starting with a capital`
            }
            placeholder="ShopDrop"
            value={draft.name}
            onChange={(name) => set({ name })}
            autoFocus={true}
          />
          <TextInput
            label={t`Folder name`}
            placeholder={draft.name}
            value={draft.repo}
            onChange={(repo) => set({ repo })}
          />
        </FieldRow>
        {noCodeOffered ? (
          <FormControlLabel
            sx={{ m: 0, alignItems: 'flex-start', gap: space.gap }}
            control={
              <Switch
                checked={draft.kind === ''}
                onChange={(_, code) => set({ kind: code ? '' : CONTENT_KIND, dependencies: null })}
              />
            }
            label={
              <Box sx={{ pt: 0.75 }}>
                <Box sx={{ fontSize: 'medium.fontSize', fontWeight: 500 }}>{t`Has code`}</Box>
                <Box sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
                  {t`A C# plugin built with the .NET SDK. Without it the mod is only the content below, and no SDK is needed.`}
                </Box>
              </Box>
            }
          />
        ) : null}
        <Params
          params={stepParams(form, 'mod')}
          all={form?.parameters ?? []}
          draft={draft}
          set={set}
        />
      </>
    )
  },
  (a, b) =>
    a.form === b.form &&
    a.noCodeOffered === b.noCodeOffered &&
    a.set === b.set &&
    a.draft.kind === b.draft.kind &&
    a.draft.name === b.draft.name &&
    a.draft.repo === b.draft.repo &&
    a.draft.values === b.draft.values,
)

function ProjectStep({
  draft,
  owners,
  set,
}: {
  draft: Draft
  owners: GitHubOwner[]
  set: SetDraft
}) {
  const { t } = useLingui()
  const profiles = useProfiles((s) => s.profiles)
  const openName = useProfiles((s) => openProfileOf(s)?.name ?? '')
  const gameName = useGameInfo()?.name ?? ''
  const folderId = useId()
  const pick = async () => {
    const dir = await PickFolder(t`Folder for your mod projects`)
    if (dir) {
      set({ parent: dir })
    }
  }
  const sep = draft.parent.includes('\\') ? '\\' : '/'
  const where = draft.parent ? `${draft.parent}${sep}${draft.repo || draft.name}` : ''
  return (
    <>
      <Field label={t`Projects folder`} htmlFor={folderId} hint={where}>
        <Box sx={{ display: 'flex', gap: space.gap }}>
          <TextField
            id={folderId}
            size="small"
            fullWidth={true}
            value={draft.parent}
            slotProps={{ input: { readOnly: true } }}
          />
          <Button variant="outlined" onClick={() => pick().catch(() => undefined)}>
            {t`Choose…`}
          </Button>
        </Box>
      </Field>
      <SelectField
        label={draft.kind ? t`Installs into` : t`Builds install into`}
        value={draft.profile}
        onChange={(profile) => set({ profile })}
      >
        {profiles.map((p) => (
          <MenuItem key={p.id} value={p.id}>
            {p.name}
          </MenuItem>
        ))}
        <MenuItem
          value={NEW_PROFILE}
        >{t`New profile: ${{ name: `${gameName} Development` }}`}</MenuItem>
      </SelectField>
      {draft.profile === NEW_PROFILE && openName ? (
        <FormControlLabel
          control={
            <Switch checked={draft.copyMods} onChange={(_, copyMods) => set({ copyMods })} />
          }
          label={t`Start it with the mods of ${{ profile: openName }}`}
        />
      ) : null}
      <RepositoryChoice draft={draft} owners={owners} set={set} />
      <SelectField
        label={t`Licence`}
        displayEmpty={true}
        value={draft.license}
        onChange={(license) => set({ license })}
      >
        <MenuItem value="">{t`MIT (recommended)`}</MenuItem>
        {LICENCES.map(([value, name]) => (
          <MenuItem key={value} value={value}>
            {name}
          </MenuItem>
        ))}
      </SelectField>
      <LicenseChart license={draft.license} />
    </>
  )
}

function useStepLabels() {
  const { t } = useLingui()
  return {
    mod: t`Mod`,
    author: t`Author`,
    features: t`Features`,
    dependencies: t`Dependencies`,
    project: t`Project`,
  }
}

// Reads the template's questions and the tools each time the dialog opens; the profile starts as the open one.
function useWizardData(open: boolean, game: string, setDraft: (fn: (d: Draft) => Draft) => void) {
  const { t } = useLingui()
  const [tools, setTools] = useState<Tools | null>(null)
  const [form, setForm] = useState<Form | null>(null)
  const [failed, setFailed] = useState(false)
  useEffect(() => {
    if (!open) {
      return
    }
    setForm(null)
    setFailed(false)
    setDraft((d) => ({ ...firstDraft(null, {}), github: d.github, owner: d.owner }))
    TemplateForm(game)
      .then((read) => {
        setForm(read)
        // What the player typed while the template loaded stays; the template fills in the rest.
        setDraft((d) => {
          const fresh = firstDraft(read, readStored(REMEMBERED, {}, isRemembered))
          const typed = Object.fromEntries(Object.entries(d.values).filter(([, v]) => v !== ''))
          return {
            ...fresh,
            name: d.name,
            repo: d.repo,
            parent: fresh.parent || d.parent,
            values: { ...fresh.values, ...typed },
            profile: useProfiles.getState().openId || NEW_PROFILE,
            github: d.github,
            owner: d.owner,
            kind: d.kind,
            dependencies: d.dependencies,
          }
        })
      })
      .catch((err: unknown) => {
        setFailed(true)
        reportError(t`Could not read the mod template`)(err)
      })
    ReadTools(game)
      .then((found) => {
        setTools(found)
        setDraft((d) => ({
          ...d,
          owner: d.owner || found.githubUser,
          parent: d.parent || found.projectsDir,
        }))
      })
      .catch(() => setTools(null))
  }, [open, game, setDraft, t])
  return { tools, form, failed }
}

export function NewModDialog({
  open,
  game,
  onClose,
}: {
  open: boolean
  game: string
  onClose: () => void
}) {
  const { t } = useLingui()
  const labels = useStepLabels()
  const [draft, setDraft] = useState<Draft>(() => firstDraft(null, {}))
  const { tools, form: templateForm, failed } = useWizardData(open, game, setDraft)
  const { form, noCodeOffered } = useWizardForm({ game, kind: draft.kind, templateForm }, setDraft)
  const steps = wizardSteps(form)
  // Kept by name: the kind and the template's form arrive late and change which steps exist.
  const [stepName, setStepName] = useState<Step | null>(null)
  const at = stepIndex(steps, stepName)
  const setAt = (i: number) => setStepName(steps[i] ?? null)
  const openId = useProfiles((s) => s.openId)
  const [pending, run] = usePending()
  useEffect(() => {
    if (open) {
      setStepName(null)
    }
  }, [open])
  const set: SetDraft = useCallback((next) => setDraft((d) => ({ ...d, ...next })), [])
  const step = steps[at] ?? 'project'
  const last = at === steps.length - 1
  let issue = last ? draftIssue(draft, form) : stepIssue(draft, form, step)
  if (failed && !draft.kind) {
    issue = t`Could not read the mod template`
  }
  const create = () =>
    run(
      async () => {
        const project = await Create({
          ...draft,
          game,
          kind: draft.kind,
          repo: draft.repo || draft.name,
          profile: draft.profile === NEW_PROFILE ? '' : draft.profile,
          copyFrom: draft.profile === NEW_PROFILE && draft.copyMods ? openId : '',
        })
        if (form) {
          writeStored(REMEMBERED, rememberedOf(draft, form))
        }
        // The mod lives in its profile: open that one, so Mods and Play show it.
        await useProfiles.getState().load(game, project.profile)
        useProfiles.getState().open(project.profile)
        useToasts
          .getState()
          .push({ kind: 'success', title: t`Started ${{ name: project.name }}`, body: project.dir })
        onClose()
        try {
          const skipped = await startDependencies(
            game,
            project.dir,
            { ids: chosenDependencies(draft, form), optional: draft.optional },
            useKnownPackages.getState(),
          )
          if (skipped.length > 0) {
            useToasts.getState().push({
              kind: 'warning',
              title: t`No version found for some dependencies, so they were not added`,
              body: skipped.join(', '),
            })
          }
        } catch (e) {
          toastError(t`Could not add the dependencies`, e)
        }
        if (!draft.kind) {
          await buildProject(game, project.dir)
        }
      },
      { errorTitle: t`Could not start the mod` },
    )
  const subtitle = [t`Step ${{ step: at + 1 }} of ${{ count: steps.length }}`, labels[step]]
    .filter(Boolean)
    .join(' · ')
  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title={t`New mod`}
      subtitle={subtitle}
      icon={ICON}
      submitLabel={last ? t`Start mod` : t`Next`}
      issue={issue}
      dirty={draft.name.trim() !== ''}
      busy={pending}
      onSubmit={last ? create : () => setAt(at + 1)}
      onBack={at > 0 ? () => setAt(at - 1) : undefined}
      closeButton={true}
    >
      {draft.kind ? null : <Prerequisites tools={tools} form={templateForm} />}
      {step === 'mod' ? (
        <ModStep draft={draft} form={form} noCodeOffered={noCodeOffered} set={set} />
      ) : null}
      {step === 'author' || step === 'features' ? (
        <Params
          params={stepParams(form, step)}
          all={form?.parameters ?? []}
          draft={draft}
          set={set}
        />
      ) : null}
      {step === 'dependencies' ? (
        <DependenciesStep game={game} profile={openId} form={form} draft={draft} set={set} />
      ) : null}
      {step === 'project' ? (
        <ProjectStep draft={draft} owners={tools?.githubOwners ?? NO_OWNERS} set={set} />
      ) : null}
    </FormDialog>
  )
}
