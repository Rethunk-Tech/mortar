import { useLingui } from '@lingui/react/macro'
import {
  Alert,
  Box,
  Button,
  Checkbox,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  MenuItem,
  Radio,
  RadioGroup,
  Step as StepItem,
  StepLabel,
  Stepper,
  TextField,
  ToggleButton,
  ToggleButtonGroup,
  Typography,
} from '@mui/material'
import { useEffect, useState } from 'react'
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
import { readStored, writeStored } from '../shell/useStoredState.ts'
import { space } from '../theme/density.ts'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { buildProject } from './builds.ts'
import {
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

function Caption({ children }: { children: string }) {
  return (
    <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary', mb: 0.5 }}>
      {children}
    </Typography>
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
    const picked = value.split('|').filter(Boolean)
    const toggle = (c: string) =>
      set((picked.includes(c) ? picked.filter((x) => x !== c) : [...picked, c]).join('|'))
    return (
      <Box>
        <Caption>{param.label}</Caption>
        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75 }}>
          {choices.map((c) => (
            <Chip
              key={c.value}
              label={c.label}
              size="small"
              color={picked.includes(c.value) ? 'primary' : 'default'}
              variant={picked.includes(c.value) ? 'filled' : 'outlined'}
              onClick={() => toggle(c.value)}
            />
          ))}
        </Box>
      </Box>
    )
  }
  if (choices.length <= TOGGLE_CHOICES) {
    return (
      <Box>
        <Caption>{param.label}</Caption>
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
      </Box>
    )
  }
  return (
    <TextField
      select={true}
      size="small"
      label={param.label}
      value={value}
      onChange={(e) => set(e.target.value)}
    >
      {choices.map((c) => (
        <MenuItem key={c.value} value={c.value}>
          {c.label}
        </MenuItem>
      ))}
    </TextField>
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
  const counter = param.maxLength > 0 ? `${value.length}/${param.maxLength}` : ''
  return (
    <TextField
      size="small"
      label={param.label}
      helperText={[param.description, counter].filter(Boolean).join(' ')}
      value={value}
      multiline={param.maxLength > 100}
      onChange={(e) => set(e.target.value)}
      sx={{ flex: 1, minWidth: 0 }}
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
      <Box sx={{ display: 'flex', gap: space.gap }}>
        <TextField
          size="small"
          label={t`Mod name`}
          helperText={t`Letters and digits, starting with a capital`}
          placeholder="ShopDrop"
          value={draft.name}
          onChange={(e) => set({ name: e.target.value })}
          autoFocus={true}
          sx={{ flex: 2 }}
        />
        <TextField
          size="small"
          label={t`Folder name`}
          placeholder={draft.name}
          value={draft.repo}
          onChange={(e) => set({ repo: e.target.value })}
          sx={{ flex: 1 }}
        />
      </Box>
      <Params params={stepParams(form, 'mod')} draft={draft} set={set} />
    </>
  )
}

function AuthorStep({ draft, form, set }: { draft: Draft; form: Form | null; set: Set }) {
  const { t } = useLingui()
  return (
    <>
      <Typography
        sx={{ color: 'text.secondary' }}
      >{t`Mortar remembers these for your next mod.`}</Typography>
      <Params params={stepParams(form, 'author')} draft={draft} set={set} />
    </>
  )
}

function SourceChoice({ draft, user, set }: { draft: Draft; user: string; set: Set }) {
  const { t } = useLingui()
  const github = sourceLabel('github')
  return (
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
      {draft.github ? (
        <Box sx={{ display: 'flex', gap: space.gap, alignItems: 'center', pl: 4 }}>
          <TextField
            size="small"
            label={t`Account or organisation`}
            value={draft.owner}
            onChange={(e) => set({ owner: e.target.value })}
            sx={{ flex: 1 }}
          />
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
    </RadioGroup>
  )
}

function ProjectStep({ draft, user, set }: { draft: Draft; user: string; set: Set }) {
  const { t } = useLingui()
  const profiles = useProfiles((s) => s.profiles)
  const gameName = useGameInfo()?.name ?? ''
  const pick = async () => {
    const dir = await PickFolder(t`Folder for your mod projects`)
    if (dir) {
      set({ parent: dir })
    }
  }
  const where = draft.parent ? `${draft.parent}/${draft.repo || draft.name}` : ''
  return (
    <>
      <Box sx={{ display: 'flex', gap: space.gap, alignItems: 'flex-start' }}>
        <TextField
          size="small"
          label={t`Projects folder`}
          value={draft.parent}
          helperText={where}
          sx={{ flex: 1 }}
          slotProps={{ input: { readOnly: true } }}
        />
        <Button onClick={() => pick().catch(() => undefined)} sx={{ mt: '2px' }}>
          {t`Choose…`}
        </Button>
      </Box>
      <TextField
        select={true}
        size="small"
        label={t`Builds install into`}
        value={draft.profile}
        onChange={(e) => set({ profile: e.target.value })}
      >
        {profiles.map((p) => (
          <MenuItem key={p.id} value={p.id}>
            {p.name}
          </MenuItem>
        ))}
        <MenuItem
          value={NEW_PROFILE}
        >{t`New profile: ${{ name: `${gameName} Development` }}`}</MenuItem>
      </TextField>
      <SourceChoice draft={draft} user={user} set={set} />
    </>
  )
}

function useStepLabels(): Record<Step, string> {
  const { t } = useLingui()
  return { mod: t`Mod`, author: t`Author`, project: t`Project` }
}

// Reads the template's questions and the tools each time the dialog opens; the profile starts as the open one.
function useWizardData(open: boolean, game: string, setDraft: (fn: (d: Draft) => Draft) => void) {
  const { t } = useLingui()
  const openId = useProfiles((s) => s.openId)
  const [tools, setTools] = useState<Tools | null>(null)
  const [form, setForm] = useState<Form | null>(null)
  useEffect(() => {
    if (!open) {
      return
    }
    setForm(null)
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
      .catch(reportError(t`Could not read the mod template`))
    ReadTools(game)
      .then((found) => {
        setTools(found)
        setDraft((d) => (d.owner || !found.githubUser ? d : { ...d, owner: found.githubUser }))
      })
      .catch(() => setTools(null))
  }, [open, game, openId, setDraft, t])
  return { tools, form }
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
  const { tools, form } = useWizardData(open, game, setDraft)
  const [at, setAt] = useState(0)
  const [pending, run] = usePending()
  useEffect(() => {
    if (open) {
      setAt(0)
    }
  }, [open])
  const set: Set = (next) => setDraft((d) => ({ ...d, ...next }))
  const step = STEPS[at] ?? 'project'
  const last = at === STEPS.length - 1
  const issue = last ? draftIssue(draft, form) : stepIssue(draft, form, step)
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
        useToasts
          .getState()
          .push({ kind: 'success', title: t`Started ${{ name: project.name }}`, body: project.dir })
        onClose()
        await buildProject(game, project.dir)
      },
      { errorTitle: t`Could not start the mod` },
    )
  return (
    <Dialog open={open} onClose={pending ? undefined : onClose} maxWidth="sm" fullWidth={true}>
      <DialogTitle>{t`New mod`}</DialogTitle>
      <DialogContent
        sx={{
          display: 'flex',
          flexDirection: 'column',
          gap: space.pad,
          pt: `${space.gap} !important`,
        }}
      >
        <Stepper activeStep={at} sx={{ mb: 1 }}>
          {STEPS.map((s) => (
            <StepItem key={s}>
              <StepLabel>{labels[s]}</StepLabel>
            </StepItem>
          ))}
        </Stepper>
        <Prerequisites tools={tools} />
        {step === 'mod' ? <ModStep draft={draft} form={form} set={set} /> : null}
        {step === 'author' ? <AuthorStep draft={draft} form={form} set={set} /> : null}
        {step === 'project' ? (
          <ProjectStep draft={draft} user={tools?.githubUser ?? ''} set={set} />
        ) : null}
      </DialogContent>
      <DialogActions sx={{ bgcolor: 'background.paper' }}>
        <Button onClick={onClose} disabled={pending}>
          {t`Cancel`}
        </Button>
        <Box sx={{ flex: 1 }} />
        {at > 0 ? (
          <Button onClick={() => setAt(at - 1)} disabled={pending}>
            {t`Back`}
          </Button>
        ) : null}
        <Button
          variant="contained"
          onClick={last ? create : () => setAt(at + 1)}
          disabled={pending || issue !== ''}
          title={issue}
        >
          {last ? t`Start mod` : t`Next`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

const isRecord = (v: unknown): v is Remembered =>
  typeof v === 'object' && v !== null && Object.values(v).every((x) => typeof x === 'string')
