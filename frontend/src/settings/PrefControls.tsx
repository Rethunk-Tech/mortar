import { useLingui } from '@lingui/react/macro'
import { MenuItem, Select, Switch, TextField } from '@mui/material'
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

export function PrefSwitch({
  checked,
  onChange,
}: {
  checked: boolean
  onChange: (on: boolean) => void
}) {
  return <Switch checked={checked} onChange={(_, on) => onChange(on)} />
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
