import { useLingui } from '@lingui/react/macro'
import {
  Box,
  ButtonBase,
  MenuItem,
  Radio,
  Select,
  Switch,
  TextField,
  ToggleButton,
  ToggleButtonGroup,
} from '@mui/material'
import { useEffect, useState } from 'react'
import { errorText } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

export function PrefSelect({
  value,
  onChange,
  options,
}: {
  value: string
  onChange: (v: string) => void
  options: { value: string; label: string }[]
}) {
  return (
    <Select
      size="small"
      value={value}
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
    <Box
      role="radiogroup"
      aria-label={label}
      sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}
    >
      {options.map((o) => {
        const on = o.value === value
        return (
          <ButtonBase
            key={o.value}
            role="radio"
            aria-checked={on}
            onClick={() => onChange(o.value)}
            sx={{
              display: 'flex',
              alignItems: 'flex-start',
              justifyContent: 'flex-start',
              textAlign: 'left',
              gap: 1.5,
              px: 1.5,
              py: 1.25,
              borderRadius: '6px',
              bgcolor: on ? 'var(--mortar-hairline-12)' : 'var(--mortar-raised)',
              outline: on ? '1px solid' : 'none',
              outlineColor: 'primary.main',
              '&:hover': { bgcolor: 'var(--mortar-hairline-12)' },
            }}
          >
            <Radio checked={on} size="small" tabIndex={-1} sx={{ p: 0, mt: '2px' }} />
            <Box sx={{ minWidth: 0 }}>
              <Box sx={{ fontSize: 15, fontWeight: 600 }}>{o.label}</Box>
              {o.hint ? <Box sx={{ fontSize: 14, color: 'text.secondary' }}>{o.hint}</Box> : null}
            </Box>
          </ButtonBase>
        )
      })}
    </Box>
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
}: {
  value: string
  onCommit: (v: string) => Promise<void>
  placeholder?: string
}) {
  const { t } = useLingui()
  const [draft, setDraft] = useState(value)
  const push = useToasts((s) => s.push)
  useEffect(() => setDraft(value), [value])
  const commit = () => {
    if (draft === value) {
      return
    }
    onCommit(draft).catch((err: unknown) => {
      const body = errorText(err)
      push({ kind: 'error', title: t`Couldn't save that setting`, ...(body ? { body } : {}) })
      setDraft(value)
    })
  }
  return (
    <TextField
      size="small"
      value={draft}
      placeholder={placeholder}
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
}: {
  value: number
  min: number
  max: number
  onCommit: (n: number) => Promise<void>
}) {
  const { t } = useLingui()
  const [draft, setDraft] = useState(String(value))
  const push = useToasts((s) => s.push)
  useEffect(() => setDraft(String(value)), [value])
  const commit = () => {
    const n = Number(draft)
    if (!Number.isInteger(n) || n < min || n > max) {
      setDraft(String(value))
      return
    }
    if (n !== value) {
      onCommit(n).catch((err: unknown) => {
        const body = errorText(err)
        push({ kind: 'error', title: t`Couldn't save that setting`, ...(body ? { body } : {}) })
        setDraft(String(value))
      })
    }
  }
  return (
    <TextField
      type="number"
      size="small"
      value={draft}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => {
        if (e.key === 'Enter' && e.target instanceof HTMLInputElement) {
          e.target.blur()
        }
      }}
      slotProps={{ htmlInput: { min, max, step: 1 } }}
      sx={{ width: 120 }}
    />
  )
}
