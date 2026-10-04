import type { I18n } from '@lingui/core'
import { MenuItem, Select, Slider, Switch, TextField, Typography } from '@mui/material'
import { type GmcmOption, optionDraft, optionKey } from './configMenu.ts'
import { menuControl } from './configMenuKinds.ts'

const text = { fontSize: 13 } as const
const hexLen = 7
const byteMax = 255
const hexRadix = 16
const sliderFallback = 100

function asNumber(v: unknown): number {
  return typeof v === 'number' ? v : Number(v)
}

function asText(v: unknown): string {
  return v === null || v === undefined ? '' : String(v)
}

function colorHex(v: unknown): string {
  if (typeof v === 'string' && v.startsWith('#')) {
    return v.slice(0, hexLen)
  }
  if (v && typeof v === 'object' && 'R' in v && 'G' in v && 'B' in v) {
    const c = v as { R: number; G: number; B: number }
    const h = (n: number) => Math.max(0, Math.min(byteMax, n)).toString(hexRadix).padStart(2, '0')
    return `#${h(c.R)}${h(c.G)}${h(c.B)}`
  }
  return '#000000'
}

function colorAllowsAlpha(v: unknown): boolean {
  if (typeof v === 'string') {
    return v.length > hexLen
  }
  return Boolean(v && typeof v === 'object' && 'A' in v)
}

function formatLabel(opt: GmcmOption, value: unknown): string {
  const samples = opt.formatSamples ?? []
  if (samples.length === 1) {
    return samples[0] ?? ''
  }
  return asText(value)
}

function NumberControl({
  opt,
  value,
  disabled,
  onChange,
}: {
  opt: GmcmOption
  value: unknown
  disabled: boolean
  onChange: (n: number) => void
}) {
  const n = asNumber(value)
  return (
    <>
      <Slider
        size="small"
        min={opt.min ?? 0}
        max={opt.max ?? sliderFallback}
        step={opt.interval ?? 1}
        value={n}
        disabled={disabled}
        aria-label={opt.name}
        getAriaValueText={(v) => formatLabel(opt, v)}
        onChange={(_, next) => onChange(next as number)}
      />
      <Typography sx={text}>{formatLabel(opt, n)}</Typography>
    </>
  )
}

function ColorControl({
  name,
  value,
  disabled,
  i18n,
  onChange,
}: {
  name: string
  value: unknown
  disabled: boolean
  i18n: I18n
  onChange: (v: unknown) => void
}) {
  const hex = colorHex(value)
  const alpha =
    typeof value === 'object' && value && 'A' in value ? (value as { A: number }).A : byteMax
  return (
    <>
      <TextField
        size="small"
        type="color"
        value={hex}
        disabled={disabled}
        slotProps={{ htmlInput: { 'aria-label': name } }}
        onChange={(e) => onChange(e.target.value)}
      />
      <TextField
        size="small"
        value={hex}
        disabled={disabled}
        slotProps={{
          htmlInput: {
            'aria-label': i18n._({ id: 'gmcm.hex', message: '{name} as hex', values: { name } }),
          },
        }}
        onChange={(e) => onChange(e.target.value)}
      />
      {colorAllowsAlpha(value) ? (
        <TextField
          size="small"
          type="number"
          label={i18n._({ id: 'gmcm.alpha', message: 'Alpha' })}
          disabled={disabled}
          value={alpha}
          onChange={(e) =>
            onChange({
              ...(typeof value === 'object' && value ? value : {}),
              A: Number(e.target.value),
            })
          }
        />
      ) : null}
    </>
  )
}

export function MenuControl({
  page,
  opt,
  drafts,
  i18n,
  onChange,
}: {
  page: string
  opt: GmcmOption
  drafts: Record<string, unknown>
  i18n: I18n
  onChange: (key: string, value: unknown) => void
}) {
  const key = optionKey(page, opt.index)
  const value = optionDraft(page, opt, drafts)
  const kind = menuControl(opt.kind)
  const disabled = !opt.editable
  if (kind === 'switch') {
    return (
      <Switch
        size="small"
        checked={Boolean(value)}
        disabled={disabled}
        slotProps={{ input: { 'aria-label': opt.name } }}
        onChange={(_, on) => onChange(key, on)}
      />
    )
  }
  if (kind === 'number') {
    return (
      <NumberControl
        opt={opt}
        value={value}
        disabled={disabled}
        onChange={(n) => onChange(key, n)}
      />
    )
  }
  if (kind === 'select' || kind === 'image') {
    return (
      <Select
        size="small"
        fullWidth={true}
        value={asText(value)}
        disabled={disabled}
        inputProps={{ 'aria-label': opt.name }}
        onChange={(e) => onChange(key, e.target.value)}
      >
        {(opt.choices ?? []).map((c) => (
          <MenuItem key={c.value} value={c.value}>
            {c.label || c.value}
          </MenuItem>
        ))}
      </Select>
    )
  }
  if (kind === 'color') {
    return (
      <ColorControl
        name={opt.name}
        value={value}
        disabled={disabled}
        i18n={i18n}
        onChange={(next) => onChange(key, next)}
      />
    )
  }
  if (kind === 'text' || kind === 'keybind') {
    return (
      <TextField
        size="small"
        fullWidth={true}
        value={asText(value)}
        disabled={disabled}
        slotProps={{ htmlInput: { 'aria-label': opt.name } }}
        onChange={(e) => onChange(key, e.target.value)}
      />
    )
  }
  return <Typography sx={text}>{asText(value)}</Typography>
}
