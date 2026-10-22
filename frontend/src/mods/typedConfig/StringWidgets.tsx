import { useLingui } from '@lingui/react/macro'
import {
  Autocomplete,
  Box,
  Button,
  Chip,
  IconButton,
  InputAdornment,
  TextField,
} from '@mui/material'
import { Circle, Plus, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import { space } from '../../theme/density.ts'
import { isNumericItem, listSeparator, splitItems } from './entries.ts'
import { clearedKeybind, formatKeypress, usesInputSystem } from './keybind.ts'
import { TEXT_WIDTH_PX, type WidgetProps } from './widgetLayout.ts'

const ITEM_WIDTH_PX = 84

// A text field you can still type into, with a Record button that takes the next key press as the whole value.
function KeybindWidget({ entry, label, onChange }: WidgetProps) {
  const { t } = useLingui()
  const [recording, setRecording] = useState(false)
  const value = String(entry.value)
  const inputSystem = usesInputSystem(value, String(entry.default))
  useEffect(() => {
    if (!recording) {
      return
    }
    const onKey = (e: KeyboardEvent) => {
      e.preventDefault()
      e.stopPropagation()
      if (e.repeat) {
        return
      }
      if (e.code === 'Escape') {
        setRecording(false)
        return
      }
      const recorded = formatKeypress(
        { code: e.code, shift: e.shiftKey, ctrl: e.ctrlKey, alt: e.altKey },
        inputSystem,
      )
      if (recorded !== null) {
        onChange(recorded)
        setRecording(false)
      }
    }
    globalThis.addEventListener('keydown', onKey, true)
    return () => globalThis.removeEventListener('keydown', onKey, true)
  }, [recording, inputSystem, onChange])
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
      <Button
        size="small"
        variant={recording ? 'contained' : 'outlined'}
        aria-label={t`Record ${{ name: label }}`}
        aria-pressed={recording}
        startIcon={<Circle size={12} />}
        onClick={() => setRecording((r) => !r)}
        onBlur={() => setRecording(false)}
      >
        {recording ? t`Press a key…` : t`Record`}
      </Button>
      <TextField
        size="small"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        slotProps={{
          htmlInput: { 'aria-label': label },
          input: {
            endAdornment: (
              <InputAdornment position="end">
                <IconButton
                  size="small"
                  edge="end"
                  aria-label={t`Clear ${{ name: label }}`}
                  onClick={() => onChange(clearedKeybind(inputSystem))}
                >
                  <X size={14} />
                </IconButton>
              </InputAdornment>
            ),
          },
        }}
        sx={{ width: TEXT_WIDTH_PX }}
      />
    </Box>
  )
}

function ChipsWidget({ entry, label, onChange }: WidgetProps) {
  const separator = listSeparator(entry.value, entry.default)
  return (
    <Autocomplete
      multiple={true}
      freeSolo={true}
      size="small"
      options={[]}
      value={splitItems(entry.value)}
      onChange={(_, items) =>
        onChange(
          items
            .map((i) => i.trim())
            .filter((i) => i !== '')
            .join(separator),
        )
      }
      renderValue={(items, getItemProps) =>
        items.map((item, i) => {
          const { key, ...itemProps } = getItemProps({ index: i })
          return <Chip key={key} size="small" label={item} {...itemProps} />
        })
      }
      renderInput={(params) => (
        <TextField
          {...params}
          slotProps={{
            ...params.slotProps,
            htmlInput: { ...params.slotProps.htmlInput, 'aria-label': label },
          }}
        />
      )}
      // The theme pins small inputs to one control height; chips wrap onto more lines, so this field grows.
      sx={{
        width: TEXT_WIDTH_PX,
        '&&& .MuiInputBase-root': { height: 'auto', minHeight: space.control, flexWrap: 'wrap' },
        // The typing area shares the chips' line and only wraps when they fill it.
        '&&& .MuiAutocomplete-input': { width: 0, minWidth: 30, flexGrow: 1 },
      }}
    />
  )
}

const positionKey = (position: number) => `number-${position}`

function NumberItem({
  value,
  label,
  onValid,
  onRemove,
}: {
  value: string
  label: string
  onValid: (value: string) => void
  onRemove: (() => void) | null
}) {
  const { t } = useLingui()
  const [draft, setDraft] = useState<string | null>(null)
  const shown = draft ?? value
  return (
    <TextField
      size="small"
      value={shown}
      error={!isNumericItem(shown)}
      onChange={(e) => {
        setDraft(e.target.value)
        if (isNumericItem(e.target.value)) {
          onValid(e.target.value)
        }
      }}
      onBlur={() => setDraft(null)}
      slotProps={{
        htmlInput: { 'aria-label': label },
        input: onRemove
          ? {
              endAdornment: (
                <InputAdornment position="end">
                  <IconButton
                    size="small"
                    edge="end"
                    aria-label={t`Remove ${{ name: label }}`}
                    onClick={onRemove}
                  >
                    <X size={14} />
                  </IconButton>
                </InputAdornment>
              ),
            }
          : {},
      }}
      sx={{ width: ITEM_WIDTH_PX }}
    />
  )
}

// A vector or number list: one field per item, written back with the separator the mod uses.
function NumberListWidget({ entry, label, onChange }: WidgetProps) {
  const { t } = useLingui()
  const items = splitItems(entry.value)
  const separator = listSeparator(entry.value, entry.default)
  const write = (next: string[]) => onChange(next.join(separator))
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: space.gap }}>
      <IconButton
        size="small"
        aria-label={t`Add ${{ name: label }} item`}
        onClick={() => write([...items, items.at(-1) ?? '0'])}
      >
        <Plus size={14} />
      </IconButton>
      {items.map((item, i) => (
        <NumberItem
          key={positionKey(i)}
          value={item}
          label={`${label} ${i + 1}`}
          onValid={(v) => write(items.map((old, j) => (j === i ? v : old)))}
          onRemove={items.length > 1 ? () => write(items.filter((_v, j) => j !== i)) : null}
        />
      ))}
    </Box>
  )
}

export { ChipsWidget, KeybindWidget, NumberListWidget }
