import {
  Box,
  FormControlLabel,
  MenuItem,
  Switch,
  ToggleButton,
  ToggleButtonGroup,
} from '@mui/material'
import type { Param } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import { Field, SelectField, TextInput } from '../shell/dialog/FormDialog.tsx'
import { space } from '../theme/density.ts'
import { CONTENT_PARAM } from './assetKinds.ts'
import { ChoicePicker } from './ChoicePicker.tsx'
import { ContentTicks } from './ContentTicks.tsx'
import { type Draft, requirementIssue } from './draft.ts'
import { localizedParam } from './ownParams.ts'

// A single choice with this many options or fewer is a row of toggle buttons rather than a menu.
const TOGGLE_CHOICES = 3

export type SetDraft = (next: Partial<Draft>) => void

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
  if (param.id === CONTENT_PARAM) {
    return <ContentTicks param={param} value={value} set={set} />
  }
  if (param.kind === 'choice') {
    return <ChoiceField param={param} value={value} set={set} />
  }
  if (param.kind === 'bool') {
    return (
      <FormControlLabel
        sx={{ m: 0, alignItems: 'flex-start', gap: space.gap }}
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

export function Params({
  params,
  all,
  draft,
  set,
}: {
  params: Param[]
  all: Param[]
  draft: Draft
  set: SetDraft
}) {
  return params.map((p) => (
    <ParamField
      key={p.id}
      param={localizedParam(p)}
      value={draft.values[p.id] ?? ''}
      // A tick changes the recommended mods, so the picks start again from them.
      set={(v) =>
        set({
          values: { ...draft.values, [p.id]: v },
          ...(p.id === CONTENT_PARAM ? { dependencies: null } : {}),
        })
      }
      blocked={p.kind === 'bool' ? requirementIssue(p, draft.values, all) : ''}
    />
  ))
}
