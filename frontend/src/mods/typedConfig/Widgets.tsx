import { useLingui } from '@lingui/react/macro'
import { Box, Button, IconButton, MenuItem, Select, Slider, Switch, TextField } from '@mui/material'
import { Plus, X } from 'lucide-react'
import { useState } from 'react'
import { space } from '../../theme/density.ts'
import { isHexColor, parseNumber, wantsSlider } from './entries.ts'
import type { ConfigEntry, ConfigValue } from './types.ts'

const NUMBER_WIDTH_PX = 120
const TEXT_WIDTH_PX = 280
const SLIDER_WIDTH_PX = 160
const SWATCH_PX = 28
const SLIDER_STEPS = 100
const RGB_LENGTH = 7
const FALLBACK_COLOR = '#000000'

interface WidgetProps {
  entry: ConfigEntry
  label: string
  onChange: (value: ConfigValue) => void
}

function NumberWidget({ entry, label, onChange }: WidgetProps) {
  const value = Number(entry.value)
  const [draft, setDraft] = useState<string | null>(null)
  const commit = (raw: string) => {
    const n = parseNumber(raw, entry)
    if (n !== null) {
      onChange(n)
    }
    setDraft(null)
  }
  const field = (
    <TextField
      size="small"
      type="number"
      value={draft ?? String(value)}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={(e) => commit(e.target.value)}
      onKeyDown={(e) => {
        if (e.key === 'Enter') {
          commit(e.currentTarget.querySelector('input')?.value ?? '')
        }
      }}
      slotProps={{
        htmlInput: {
          'aria-label': label,
          ...(entry.min === undefined ? {} : { min: entry.min }),
          ...(entry.max === undefined ? {} : { max: entry.max }),
          step: entry.type === 'int' ? 1 : 'any',
        },
      }}
      sx={{ width: NUMBER_WIDTH_PX }}
    />
  )
  if (!wantsSlider(entry)) {
    return field
  }
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: space.pad }}>
      <Slider
        size="small"
        aria-label={label}
        value={value}
        min={entry.min ?? 0}
        max={entry.max ?? 0}
        step={entry.type === 'int' ? 1 : ((entry.max ?? 1) - (entry.min ?? 0)) / SLIDER_STEPS}
        onChange={(_e, v) => onChange(typeof v === 'number' ? v : value)}
        sx={{ width: SLIDER_WIDTH_PX }}
      />
      {field}
    </Box>
  )
}

function ColorWidget({ entry, label, onChange }: WidgetProps) {
  const value = String(entry.value)
  const hashed = value.startsWith('#') ? value : `#${value}`
  const normalized = isHexColor(value) ? hashed : FALLBACK_COLOR
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
      <Box
        component="input"
        type="color"
        aria-label={label}
        value={normalized.slice(0, RGB_LENGTH)}
        onChange={(e) => onChange(e.target.value)}
        sx={{
          width: SWATCH_PX,
          height: SWATCH_PX,
          p: 0,
          border: '1px solid var(--mortar-hairline)',
          borderRadius: '4px',
          bgcolor: 'transparent',
        }}
      />
      <TextField
        size="small"
        value={value}
        error={!isHexColor(value)}
        onChange={(e) => onChange(e.target.value)}
        slotProps={{ htmlInput: { 'aria-label': tHex(label) } }}
        sx={{ width: NUMBER_WIDTH_PX }}
      />
    </Box>
  )
}

const tHex = (label: string) => `${label} hex`

// A list's rows are edited in place, so a row is told apart by its position.
const itemKey = (position: number) => `item-${position}`

function ListWidget({ entry, label, onChange }: WidgetProps) {
  const { t } = useLingui()
  const items = Array.isArray(entry.value) ? entry.value : []
  const [draft, setDraft] = useState('')
  const add = () => {
    if (draft.trim() !== '') {
      onChange([...items, draft.trim()])
      setDraft('')
    }
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, alignItems: 'flex-end' }}>
      {items.map((item, i) => (
        <Box key={itemKey(i)} sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
          <TextField
            size="small"
            value={item}
            onChange={(e) => onChange(items.map((v, j) => (j === i ? e.target.value : v)))}
            slotProps={{ htmlInput: { 'aria-label': `${label} ${i + 1}` } }}
            sx={{ width: TEXT_WIDTH_PX }}
          />
          <IconButton
            size="small"
            aria-label={t`Remove ${{ name: item }}`}
            onClick={() => onChange(items.filter((_v, j) => j !== i))}
          >
            <X size={14} />
          </IconButton>
        </Box>
      ))}
      <Box sx={{ display: 'flex', gap: 0.5 }}>
        <TextField
          size="small"
          value={draft}
          placeholder={t`Add an item`}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              add()
            }
          }}
          slotProps={{ htmlInput: { 'aria-label': t`New ${label} item` } }}
          sx={{ width: TEXT_WIDTH_PX }}
        />
        <Button size="small" startIcon={<Plus size={14} />} onClick={add}>
          {t`Add`}
        </Button>
      </Box>
    </Box>
  )
}

function EntryWidget(props: WidgetProps) {
  const { entry, label, onChange } = props
  switch (entry.type) {
    case 'bool':
      return (
        <Switch
          checked={entry.value === true}
          onChange={(e) => onChange(e.target.checked)}
          slotProps={{ input: { 'aria-label': label } }}
        />
      )
    case 'int':
    case 'float':
      return <NumberWidget {...props} />
    case 'enum':
      return (
        <Select
          size="small"
          value={String(entry.value)}
          onChange={(e) => onChange(e.target.value)}
          inputProps={{ 'aria-label': label }}
          sx={{ minWidth: NUMBER_WIDTH_PX }}
        >
          {(entry.options ?? []).map((o, i) => (
            <MenuItem key={o} value={o}>
              {entry.optionLabels?.[i] || o}
            </MenuItem>
          ))}
        </Select>
      )
    case 'color':
      return <ColorWidget {...props} />
    case 'list':
      return <ListWidget {...props} />
    default:
      return (
        <TextField
          size="small"
          value={String(entry.value)}
          onChange={(e) => onChange(e.target.value)}
          slotProps={{ htmlInput: { 'aria-label': label } }}
          sx={{ width: TEXT_WIDTH_PX }}
        />
      )
  }
}

export { EntryWidget }
