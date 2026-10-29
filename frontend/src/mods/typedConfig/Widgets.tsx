import { useLingui } from '@lingui/react/macro'
import {
  Autocomplete,
  Box,
  Button,
  ButtonBase,
  IconButton,
  MenuItem,
  Popover,
  Select,
  Slider,
  Switch,
  TextField,
} from '@mui/material'
import { Plus, X } from 'lucide-react'
import { useState } from 'react'
import { HexColorPicker } from 'react-colorful'
import { labelledInput } from '../../shell/labelledInput.tsx'
import { space } from '../../theme/density.ts'
import {
  commaListKind,
  formatLike,
  isHexColor,
  looksLikeColour,
  parseNumber,
  suggestedOptions,
  wantsSlider,
} from './entries.ts'
import { looksLikeKeybind } from './keybind.ts'
import { ChipsWidget, KeybindWidget, NumberListWidget } from './StringWidgets.tsx'
import type { ConfigValue } from './types.ts'
import { NUMBER_WIDTH_PX, TEXT_WIDTH_PX, type WidgetProps } from './widgetLayout.ts'

const SLIDER_WIDTH_PX = 160
const SWATCH_PX = 28
const SLIDER_STEPS = 100
const RGB_LENGTH = 7
const FALLBACK_COLOR = '#000000'

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
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const rgb = normalized.slice(0, RGB_LENGTH)
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
      <ButtonBase
        aria-label={label}
        onClick={(e) => setAnchor(e.currentTarget)}
        sx={{
          width: SWATCH_PX,
          height: SWATCH_PX,
          border: '1px solid var(--mortar-hairline)',
          borderRadius: '4px',
          bgcolor: rgb,
        }}
      />
      <Popover
        open={anchor !== null}
        anchorEl={anchor}
        onClose={() => setAnchor(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
        // The picker's round handles reach past its edges; padding keeps them inside rather than scrolling.
        slotProps={{ paper: { sx: { p: 1.5, overflow: 'visible' } } }}
      >
        <HexColorPicker
          color={rgb}
          onChange={(picked) =>
            onChange(formatLike(picked, isHexColor(value) ? value : String(entry.default)))
          }
        />
      </Popover>
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
          slotProps={{ htmlInput: { 'aria-label': t`New ${{ name: label }} item` } }}
          sx={{ width: TEXT_WIDTH_PX }}
        />
        <Button startIcon={<Plus size={14} />} onClick={add}>
          {t`Add`}
        </Button>
      </Box>
    </Box>
  )
}

const splitMembers = (value: ConfigValue): string[] =>
  String(value)
    .split(',')
    .map((m) => m.trim())
    .filter((m) => m !== '')

function MultiWidget({ entry, label, onChange }: WidgetProps) {
  return (
    <Select
      multiple={true}
      size="small"
      value={splitMembers(entry.value)}
      onChange={(e) =>
        onChange(
          (typeof e.target.value === 'string' ? [e.target.value] : e.target.value).join(', '),
        )
      }
      inputProps={{ 'aria-label': label }}
      sx={{ minWidth: TEXT_WIDTH_PX }}
    >
      {(entry.options ?? []).map((o) => (
        <MenuItem key={o} value={o}>
          {o}
        </MenuItem>
      ))}
    </Select>
  )
}

// Free text, offering the values the description lists while still taking any other.
function TextWidget(props: WidgetProps) {
  const { entry, label, onChange } = props
  if (looksLikeColour(entry)) {
    return <ColorWidget {...props} />
  }
  if (looksLikeKeybind(entry)) {
    return <KeybindWidget {...props} />
  }
  const listKind = commaListKind(entry)
  if (listKind === 'numbers') {
    return <NumberListWidget {...props} />
  }
  if (listKind === 'chips') {
    return <ChipsWidget {...props} />
  }
  const options = suggestedOptions(entry.description)
  if (options.length === 0) {
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
  return (
    <Autocomplete
      freeSolo={true}
      size="small"
      options={options}
      inputValue={String(entry.value)}
      onInputChange={(_, value, reason) => {
        if (reason !== 'reset' || value !== String(entry.value)) {
          onChange(value)
        }
      }}
      renderInput={labelledInput(label)}
      sx={{ width: TEXT_WIDTH_PX }}
    />
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
    case 'multi':
      return <MultiWidget {...props} />
    case 'string':
    case 'raw':
      return <TextWidget {...props} />
    default:
      return entry.type satisfies never
  }
}

export { EntryWidget }
