import { useLingui } from '@lingui/react/macro'
import {
  Alert,
  Box,
  Button,
  Checkbox,
  FormControlLabel,
  MenuItem,
  Radio,
  RadioGroup,
  Switch,
  TextField,
  ToggleButton,
  ToggleButtonGroup,
} from '@mui/material'
import { Hammer } from 'lucide-react'
import { useEffect, useId, useState } from 'react'
import type {
  Form,
  Param,
  Tools,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  Create,
  Tools as ReadTools,
  TemplateForm,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { PickFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
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
import { type Known, startDependencies } from './dependencies.ts'
import {
  chosenDependencies,
  type Draft,
  draftIssue,
  firstDraft,
  REMEMBERED,
  type Remembered,
  rememberedOf,
  STEPS,
  type Step,
  stepIssue,
  stepParams,
} from './draft.ts'

// The profile picker's value for a new development profile; Create takes it as no profile.
const NEW_PROFILE = 'new'
// A text parameter allowed longer than this gets a box of several lines.
const MULTILINE_OVER = 100
// A single choice with this many options or fewer is a row of toggle buttons rather than a menu.
const TOGGLE_CHOICES = 3

type Set = (next: Partial<Draft>) => void

function Prerequisites({ tools }: { tools: Tools | null }) {
  const { t } = useLingui()
  if (!tools) {
    return null
  }
  const missing = [
    (tools.dotnetSdks ?? []).length === 0 ? t`Install the .NET SDK to build mods.` : '',
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
}

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
}: {
  param: Param
  value: string
  set: (v: string) => void
}) {
  if (param.kind === 'choice') {
    return <ChoiceField param={param} value={value} set={set} />
  }
  if (param.kind === 'bool') {
    return (
      <FormControlLabel
        sx={{ m: 0, alignItems: 'flex-start', gap: 1 }}
        control={<Switch checked={value === 'true'} onChange={(_, on) => set(String(on))} />}
        label={
          <Box sx={{ pt: 0.75 }}>
            <Box sx={{ fontSize: 'medium.fontSize', fontWeight: 500 }}>{param.label}</Box>
            <Box sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
              {param.description}
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
      multiline={param.maxLength > MULTILINE_OVER}
      {...(param.maxLength > 0 ? { maxLength: param.maxLength } : {})}
    />
  )
}

function Params({ params, draft, set }: { params: Param[]; draft: Draft; set: Set }) {
  return params.map((p) => (
    <ParamField
      key={p.id}
      param={p}
      value={draft.values[p.id] ?? ''}
      set={(v) => set({ values: { ...draft.values, [p.id]: v } })}
    />
  ))
}

function ModStep({ draft, form, set }: { draft: Draft; form: Form | null; set: Set }) {
  const { t } = useLingui()
  return (
    <>
      <FieldRow>
        <TextInput
          label={t`Mod name`}
          hint={t`Letters and digits, starting with a capital`}
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
      <Params params={stepParams(form, 'mod')} draft={draft} set={set} />
    </>
  )
}

function SourceChoice({ draft, user, set }: { draft: Draft; user: string; set: Set }) {
  const { t } = useLingui()
  const github = sourceLabel('github')
  return (
    <Field label={t`Repository`}>
      <RadioGroup
        value={draft.github ? 'github' : 'copy'}
        onChange={(_, v) => set({ github: v === 'github' })}
      >
        <FormControlLabel
          value="copy"
          control={<Radio size="small" />}
          label={t`Keep it on this computer`}
        />
        <FormControlLabel
          value="github"
          disabled={user === ''}
          control={<Radio size="small" />}
          label={
            user
              ? t`Also create a repository on ${{ source: github }}`
              : t`Sign in with gh to create a repository on ${{ source: github }}`
          }
        />
      </RadioGroup>
      {draft.github ? (
        <Box sx={{ display: 'flex', gap: 2, alignItems: 'flex-end', pl: 4, mt: 1 }}>
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <TextInput
              label={t`Account or organisation`}
              value={draft.owner}
              onChange={(owner) => set({ owner })}
            />
          </Box>
          <FormControlLabel
            control={
              <Checkbox
                size="small"
                checked={draft.private}
                onChange={(_, on) => set({ private: on })}
              />
            }
            label={t`Private`}
          />
        </Box>
      ) : null}
    </Field>
  )
}

function ProjectStep({ draft, user, set }: { draft: Draft; user: string; set: Set }) {
  const { t } = useLingui()
  const profiles = useProfiles((s) => s.profiles)
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
      <SourceChoice draft={draft} user={user} set={set} />
    </>
  )
}

function useStepLabels(): Record<Step, string> {
  const { t } = useLingui()
  return { mod: t`Mod`, author: t`Author`, dependencies: t`Dependencies`, project: t`Project` }
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
    TemplateForm(game)
      .then((read) => {
        setForm(read)
        setDraft((d) => ({
          ...firstDraft(read, readStored(REMEMBERED, {}, isRecord)),
          profile: openId || NEW_PROFILE,
          github: d.github,
          owner: d.owner,
        }))
      })
      .catch((err: unknown) => {
        setFailed(true)
        reportError(t`Could not read the mod template`)(err)
      })
    ReadTools(game)
      .then((found) => {
        setTools(found)
        setDraft((d) => (d.owner || !found.githubUser ? d : { ...d, owner: found.githubUser }))
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
  const [known, setKnown] = useState<Known>({})
  const openId = useProfiles((s) => s.openId)
  const [pending, run] = usePending()
  useEffect(() => {
    if (open) {
      setAt(0)
    }
  }, [open])
  const set: Set = (next) => setDraft((d) => ({ ...d, ...next }))
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
        })
        if (form) {
          writeStored(REMEMBERED, rememberedOf(draft, form))
        }
        // The mod lives in its profile: open that one, so Mods and Play show it.
        await useProfiles.getState().load(game, project.profile)
        useProfiles.getState().open(project.profile)
        await startDependencies(game, project.dir, chosenDependencies(draft, form), known)
        useToasts
          .getState()
          .push({ kind: 'success', title: t`Started ${{ name: project.name }}`, body: project.dir })
        onClose()
        await buildProject(game, project.dir)
      },
      { errorTitle: t`Could not start the mod` },
    )
  const subtitle = [
    t`Step ${{ step: at + 1 }} of ${{ count: STEPS.length }}`,
    labels[step],
    step === 'author' ? t`Mortar remembers these for your next mod.` : '',
  ]
    .filter(Boolean)
    .join(' · ')
  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title={t`New mod`}
      subtitle={subtitle}
      icon={
        <ToneIcon tone="neutral">
          <Hammer size={22} />
        </ToneIcon>
      }
      submitLabel={last ? t`Start mod` : t`Next`}
      issue={issue}
      dirty={draft.name.trim() !== ''}
      busy={pending}
      onSubmit={last ? create : () => setAt(at + 1)}
      onBack={at > 0 ? () => setAt(at - 1) : undefined}
    >
      <Prerequisites tools={tools} />
      {step === 'mod' ? <ModStep draft={draft} form={form} set={set} /> : null}
      {step === 'author' ? (
        <Params params={stepParams(form, 'author')} draft={draft} set={set} />
      ) : null}
      {step === 'dependencies' ? (
        <DependenciesStep
          game={game}
          profile={openId}
          form={form}
          draft={draft}
          known={known}
          setKnown={setKnown}
          set={set}
        />
      ) : null}
      {step === 'project' ? (
        <ProjectStep draft={draft} user={tools?.githubUser ?? ''} set={set} />
      ) : null}
    </FormDialog>
  )
}

const isRecord = (v: unknown): v is Remembered =>
  typeof v === 'object' && v !== null && Object.values(v).every((x) => typeof x === 'string')
