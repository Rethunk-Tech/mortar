import { useLingui } from '@lingui/react/macro'
import {
  Box,
  FormControlLabel,
  MenuItem,
  Radio,
  RadioGroup,
  Select,
  Switch,
  TextField,
  ToggleButton,
  ToggleButtonGroup,
} from '@mui/material'
import { useEffect, useState } from 'react'
import { reportError } from '../toasts/report.ts'

export function PrefSelect({
  value,
  onChange,
  options,
  label,
}: {
  value: string
  onChange: (v: string) => void
  options: { value: string; label: string }[]
  label: string
}) {
  return (
    <Select
      size="small"
      displayEmpty={true}
      value={value}
      inputProps={{ 'aria-label': label }}
      onChange={(e) => onChange(String(e.target.value))}
      sx={{ minWidth: 180 }}
    >
      {options.map((o) => (
        <MenuItem key={o.value} value={o.value}>
          {o.label}
        </MenuItem>
      ))}
    </Select>
  )
}

export function PrefSegmented({
  value,
  onChange,
  options,
  label,
}: {
  value: string
  onChange: (v: string) => void
  options: { value: string; label: string }[]
  label: string
}) {
  return (
    <ToggleButtonGroup
      exclusive={true}
      size="small"
      value={value}
      aria-label={label}
      onChange={(_, next: string | null) => {
        if (next !== null) {
          onChange(next)
        }
      }}
      sx={{
        bgcolor: 'var(--mortar-raised)',
        borderRadius: '6px',
        p: '3px',
        gap: '3px',
        '& .MuiToggleButton-root': {
          border: 0,
          borderRadius: '4px !important',
          px: 1.75,
          color: 'text.secondary',
        },
        '& .MuiToggleButton-root.Mui-selected, & .MuiToggleButton-root.Mui-selected:hover': {
          bgcolor: 'primary.main',
          color: 'primary.contrastText',
        },
      }}
    >
      {options.map((o) => (
        <ToggleButton key={o.value} value={o.value}>
          {o.label}
        </ToggleButton>
      ))}
    </ToggleButtonGroup>
  )
}

export function PrefCards({
  value,
  onChange,
  options,
  label,
}: {
  value: string
  onChange: (v: string) => void
  options: { value: string; label: string; hint?: string }[]
  label: string
}) {
  return (
    <RadioGroup
      aria-label={label}
      value={value}
      onChange={(_, v) => onChange(v)}
      sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}
    >
      {options.map((o) => (
        <FormControlLabel
          key={o.value}
          value={o.value}
          control={<Radio size="small" sx={{ p: 0, mt: '2px' }} />}
          label={
            <Box sx={{ minWidth: 0 }}>
              <Box sx={{ fontSize: 15, fontWeight: 600 }}>{o.label}</Box>
              {o.hint ? <Box sx={{ fontSize: 14, color: 'text.secondary' }}>{o.hint}</Box> : null}
            </Box>
          }
          sx={{
            m: 0,
            alignItems: 'flex-start',
            gap: 1.5,
            px: 1.5,
            py: 1.25,
            borderRadius: '6px',
            bgcolor: 'var(--mortar-raised)',
            '&:hover': { bgcolor: 'var(--mortar-hairline-12)' },
            '&:has(input:checked)': {
              bgcolor: 'var(--mortar-hairline-12)',
              outline: '1px solid',
              outlineColor: 'primary.main',
            },
          }}
        />
      ))}
    </RadioGroup>
  )
}

export function PrefSwitch({
  checked,
  onChange,
  label,
}: {
  checked: boolean
  onChange: (on: boolean) => void
  label?: string
}) {
  return (
    <Switch
      checked={checked}
      onChange={(_, on) => onChange(on)}
      {...(label ? { slotProps: { input: { 'aria-label': label } } } : {})}
    />
  )
}

export function PrefText({
  value,
  onCommit,
  placeholder,
  label,
}: {
  value: string
  onCommit: (v: string) => Promise<void>
  placeholder?: string
  label: string
}) {
  const { t } = useLingui()
  const [draft, setDraft] = useState(value)
  useEffect(() => setDraft(value), [value])
  const commit = () => {
    if (draft === value) {
      return
    }
    onCommit(draft).catch((err: unknown) => {
      reportError(t`Could not save that setting`)(err)
      setDraft(value)
    })
  }
  return (
    <TextField
      size="small"
      value={draft}
      placeholder={placeholder}
      slotProps={{ htmlInput: { 'aria-label': label } }}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => {
        if (e.key === 'Enter' && e.target instanceof HTMLInputElement) {
          e.target.blur()
        }
      }}
      sx={{ minWidth: 180 }}
    />
  )
}

export function PrefNumber({
  value,
  min,
  max,
  onCommit,
  label,
  disabled = false,
}: {
  value: number
  min: number
  max: number
  onCommit: (n: number) => Promise<void>
  label: string
  disabled?: boolean
}) {
  const { t } = useLingui()
  const [draft, setDraft] = useState(String(value))
  const [invalid, setInvalid] = useState(false)
  useEffect(() => setDraft(String(value)), [value])
  const range = t`Enter a number from ${min} to ${max}`
  const commit = () => {
    const n = Number(draft)
    if (!Number.isInteger(n) || n < min || n > max) {
      setInvalid(true)
      return
    }
    setInvalid(false)
    if (n !== value) {
      onCommit(n).catch((err: unknown) => {
        reportError(t`Could not save that setting`)(err)
        setDraft(String(value))
      })
    }
  }
  return (
    <TextField
      type="number"
      size="small"
      value={draft}
      error={invalid}
      disabled={disabled}
      helperText={invalid ? range : undefined}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => {
        if (e.key === 'Enter' && e.target instanceof HTMLInputElement) {
          e.target.blur()
        }
      }}
      slotProps={{ htmlInput: { min, max, step: 1, 'aria-label': label } }}
      sx={{ width: 120 }}
    />
  )
}
