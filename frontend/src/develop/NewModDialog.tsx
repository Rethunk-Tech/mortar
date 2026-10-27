import { useLingui } from '@lingui/react/macro'
import {
  Alert,
  Box,
  Button,
  FormControlLabel,
  MenuItem,
  Switch,
  TextField,
  ToggleButton,
  ToggleButtonGroup,
} from '@mui/material'
import { Hammer } from 'lucide-react'
import { memo, useCallback, useEffect, useId, useState } from 'react'
import type {
  Form,
  GitHubOwner,
  Param,
  Tools,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  Create,
  Tools as ReadTools,
  TemplateForm,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { PickFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { useGameInfo } from '../games/info.ts'
import { useProfiles } from '../profiles/store.ts'
import { ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { Field, FieldRow, FormDialog, SelectField, TextInput } from '../shell/dialog/FormDialog.tsx'
import { readStored, writeStored } from '../shell/useStoredState.ts'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { buildProject } from './builds.ts'
import { ChoicePicker } from './ChoicePicker.tsx'
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
  requirementIssue,
  STEPS,
  type Step,
  stepIssue,
  stepParams,
  validModName,
} from './draft.ts'
import { LicenseChart } from './LicenseChart.tsx'
import { RepositoryChoice } from './RepositoryChoice.tsx'
import { sdkSatisfies } from './sdk.ts'

// The profile picker's value for a new development profile; Create takes it as no profile.
const NEW_PROFILE = 'new'
// A single choice with this many options or fewer is a row of toggle buttons rather than a menu.
const TOGGLE_CHOICES = 3

type Set = (next: Partial<Draft>) => void

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

function ChoiceField({
  param,
  value,
  set,
}: {
  param: Param
  value: string
  set: (v: string) => void
}) {
  const choices = param.choices ?? []
  if (param.multiple) {
    return (
      <ChoicePicker
        label={param.label}
        helperText={param.description}
        choices={choices}
        value={value.split('|').filter(Boolean)}
        onChange={(next) => set(next.join('|'))}
      />
    )
  }
  if (choices.length <= TOGGLE_CHOICES) {
    return (
      <Field label={param.label} hint={param.description}>
        <ToggleButtonGroup
          exclusive={true}
          size="small"
          value={value}
          aria-label={param.label}
          onChange={(_, next: string | null) => next && set(next)}
        >
          {choices.map((c) => (
            <ToggleButton key={c.value} value={c.value}>
              {c.label}
            </ToggleButton>
          ))}
        </ToggleButtonGroup>
      </Field>
    )
  }
  return (
    <SelectField label={param.label} value={value} onChange={set} hint={param.description}>
      {choices.map((c) => (
        <MenuItem key={c.value} value={c.value}>
          {c.label}
        </MenuItem>
      ))}
    </SelectField>
  )
}

function ParamField({
  param,
  value,
  set,
  blocked,
}: {
  param: Param
  value: string
  set: (v: string) => void
  // Why the switch cannot be turned on with the other answers as they are.
  blocked: string
}) {
  if (param.kind === 'choice') {
    return <ChoiceField param={param} value={value} set={set} />
  }
  if (param.kind === 'bool') {
    return (
      <FormControlLabel
        sx={{ m: 0, alignItems: 'flex-start', gap: 1 }}
        control={
          <Switch
            checked={value === 'true'}
            disabled={blocked !== '' && value !== 'true'}
            onChange={(_, on) => set(String(on))}
          />
        }
        label={
          <Box sx={{ pt: 0.75 }}>
            <Box sx={{ fontSize: 'medium.fontSize', fontWeight: 500 }}>{param.label}</Box>
            <Box sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
              {blocked || param.description}
            </Box>
          </Box>
        }
      />
    )
  }
  return (
    <TextInput
      label={param.label}
      hint={param.description}
      value={value}
      onChange={set}
      {...(param.maxLength > 0 ? { maxLength: param.maxLength } : {})}
    />
  )
}

function Params({
  params,
  all,
  draft,
  set,
}: {
  params: Param[]
  all: Param[]
  draft: Draft
  set: Set
}) {
  return params.map((p) => (
    <ParamField
      key={p.id}
      param={p}
      value={draft.values[p.id] ?? ''}
      set={(v) => set({ values: { ...draft.values, [p.id]: v } })}
      blocked={p.kind === 'bool' ? requirementIssue(p, draft.values, all) : ''}
    />
  ))
}

// Memoised on what it shows, so the GitHub owner filled in as the tools arrive leaves the fields alone.
const ModStep = memo(
  function ModStepView({ draft, form, set }: { draft: Draft; form: Form | null; set: Set }) {
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
    a.set === b.set &&
    a.draft.name === b.draft.name &&
    a.draft.repo === b.draft.repo &&
    a.draft.values === b.draft.values,
)

function ProjectStep({ draft, owners, set }: { draft: Draft; owners: GitHubOwner[]; set: Set }) {
  const { t } = useLingui()
  const profiles = useProfiles((s) => s.profiles)
  const openName = useProfiles((s) => s.profiles.find((p) => p.id === s.openId)?.name ?? '')
  const gameName = useGameInfo()?.name ?? ''
  const folderId = useId()
  const pick = async () => {
    const dir = await PickFolder(t`Folder for your mod projects`)
    if (dir) {
      set({ parent: dir })
    }
  }
  const where = draft.parent ? `${draft.parent}/${draft.repo || draft.name}` : ''
  return (
    <>
      <Field label={t`Projects folder`} htmlFor={folderId} hint={where}>
        <Box sx={{ display: 'flex', gap: 1 }}>
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
        label={t`Builds install into`}
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
        label={t`License`}
        displayEmpty={true}
        value={draft.license}
        onChange={(license) => set({ license })}
      >
        <MenuItem value="">{t`MIT (Recommended)`}</MenuItem>
        <MenuItem value="apache-2.0">Apache 2.0</MenuItem>
        <MenuItem value="gpl-3.0">GNU GPL 3.0</MenuItem>
        <MenuItem value="mpl-2.0">Mozilla Public License 2.0</MenuItem>
        <MenuItem value="unlicense">The Unlicense</MenuItem>
      </SelectField>
      <LicenseChart license={draft.license} />
    </>
  )
}

function useStepLabels(): Record<Step, string> {
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
  const openId = useProfiles((s) => s.openId)
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
            profile: openId || NEW_PROFILE,
            github: d.github,
            owner: d.owner,
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
  }, [open, game, openId, setDraft, t])
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
  const { tools, form, failed } = useWizardData(open, game, setDraft)
  const [at, setAt] = useState(0)
  const openId = useProfiles((s) => s.openId)
  const [pending, run] = usePending()
  useEffect(() => {
    if (open) {
      setAt(0)
    }
  }, [open])
  const set: Set = useCallback((next) => setDraft((d) => ({ ...d, ...next })), [])
  const step = STEPS[at] ?? 'project'
  const last = at === STEPS.length - 1
  let issue = last ? draftIssue(draft, form) : stepIssue(draft, form, step)
  if (failed) {
    issue = t`Could not read the mod template`
  }
  const create = () =>
    run(
      async () => {
        const project = await Create({
          ...draft,
          game,
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
        await startDependencies(
          game,
          project.dir,
          { ids: chosenDependencies(draft, form), optional: draft.optional },
          useKnownPackages.getState(),
        )
        useToasts
          .getState()
          .push({ kind: 'success', title: t`Started ${{ name: project.name }}`, body: project.dir })
        onClose()
        await buildProject(game, project.dir)
      },
      { errorTitle: t`Could not start the mod` },
    )
  const subtitle = [t`Step ${{ step: at + 1 }} of ${{ count: STEPS.length }}`, labels[step]]
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
      <Prerequisites tools={tools} form={form} />
      {step === 'mod' ? <ModStep draft={draft} form={form} set={set} /> : null}
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
