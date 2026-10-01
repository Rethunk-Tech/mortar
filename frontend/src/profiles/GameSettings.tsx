import { useLingui } from '@lingui/react/macro'
import {
  FormControl,
  FormControlLabel,
  InputLabel,
  MenuItem,
  Select,
  Slider,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material'
import { useId } from 'react'

const PERCENT_MIN = 0
const PERCENT_MAX = 100
const PERCENT_DEFAULT = 100

interface GameSettingsProps {
  value: GameSettingsValues | null
  onChange: (value: GameSettingsValues | null) => void
}

interface NumberFieldProps {
  disabled: boolean
  label: string
  value: number | ''
  onChange: (value: number | undefined) => void
}

function NumberField({ disabled, label, value, onChange }: NumberFieldProps) {
  return (
    <TextField
      disabled={disabled}
      fullWidth={true}
      label={label}
      type="number"
      value={value}
      onChange={(event) =>
        onChange(event.target.value === '' ? undefined : Number(event.target.value))
      }
    />
  )
}

interface WindowModeFieldProps {
  disabled: boolean
  value: WindowMode | undefined
  onChange: (value: WindowMode | undefined) => void
}

function WindowModeField({ disabled, value, onChange }: WindowModeFieldProps) {
  const { t } = useLingui()
  const labelId = useId()
  return (
    <FormControl disabled={disabled} fullWidth={true}>
      <InputLabel id={labelId}>{t`Window mode`}</InputLabel>
      <Select
        labelId={labelId}
        label={t`Window mode`}
        value={value ?? ''}
        onChange={(event) => {
          const selected = String(event.target.value)
          onChange(selected === '' ? undefined : (selected as WindowMode))
        }}
      >
        <MenuItem value="">{t`Use normal setting`}</MenuItem>
        <MenuItem value="windowed">{t`Windowed`}</MenuItem>
        <MenuItem value="fullscreen">{t`Fullscreen`}</MenuItem>
        <MenuItem value="borderless">{t`Borderless`}</MenuItem>
      </Select>
    </FormControl>
  )
}

type PercentageFieldKind = 'zoom' | 'uiScale' | 'music' | 'sound'

interface PercentageFieldProps {
  disabled: boolean
  kind: PercentageFieldKind
  value: number
  onChange: (value: number) => void
}

function PercentageField({ disabled, kind, value, onChange }: PercentageFieldProps) {
  const { t } = useLingui()
  let label = t`Sound volume (${value}%)`
  if (kind === 'zoom') {
    label = t`Zoom (${value}%)`
  } else if (kind === 'uiScale') {
    label = t`UI scale (${value}%)`
  } else if (kind === 'music') {
    label = t`Music volume (${value}%)`
  }
  return (
    <Stack spacing={1}>
      <Typography>{label}</Typography>
      <Slider
        disabled={disabled}
        min={PERCENT_MIN}
        max={PERCENT_MAX}
        value={value}
        valueLabelDisplay="auto"
        onChange={(_, nextValue) => {
          if (typeof nextValue === 'number') {
            onChange(nextValue)
          }
        }}
      />
    </Stack>
  )
}

export function GameSettings({ value, onChange }: GameSettingsProps) {
  const { t } = useLingui()
  const disabled = value === null

  function updateField<K extends keyof GameSettingsValues>(
    field: K,
    fieldValue: GameSettingsValues[K],
  ) {
    onChange({ ...(value ?? {}), [field]: fieldValue })
  }

  function clearField<K extends keyof GameSettingsValues>(field: K) {
    if (value === null) {
      return
    }
    const next = { ...value }
    delete next[field]
    onChange(Object.keys(next).length === 0 ? null : next)
  }

  function numberValue(field: keyof GameSettingsValues) {
    const fieldValue = value?.[field]
    return typeof fieldValue === 'number' ? fieldValue : ''
  }

  return (
    <Stack spacing={2}>
      <Typography variant="h6">{t`Game settings`}</Typography>
      <FormControlLabel
        control={
          <Switch
            checked={disabled}
            onChange={(event) => onChange(event.target.checked ? null : {})}
          />
        }
        label={t`Use my normal settings`}
      />
      <WindowModeField
        disabled={disabled}
        value={value?.windowMode}
        onChange={(next) =>
          next === undefined ? clearField('windowMode') : updateField('windowMode', next)
        }
      />
      <NumberField
        disabled={disabled}
        label={t`Display`}
        value={numberValue('displayIndex')}
        onChange={(next) =>
          next === undefined ? clearField('displayIndex') : updateField('displayIndex', next)
        }
      />
      <Stack direction="row" spacing={2}>
        <NumberField
          disabled={disabled}
          label={t`Windowed resolution width`}
          value={numberValue('preferredResolutionX')}
          onChange={(next) =>
            next === undefined
              ? clearField('preferredResolutionX')
              : updateField('preferredResolutionX', next)
          }
        />
        <NumberField
          disabled={disabled}
          label={t`Windowed resolution height`}
          value={numberValue('preferredResolutionY')}
          onChange={(next) =>
            next === undefined
              ? clearField('preferredResolutionY')
              : updateField('preferredResolutionY', next)
          }
        />
      </Stack>
      <Stack direction="row" spacing={2}>
        <NumberField
          disabled={disabled}
          label={t`Fullscreen resolution width`}
          value={numberValue('fullscreenResolutionX')}
          onChange={(next) =>
            next === undefined
              ? clearField('fullscreenResolutionX')
              : updateField('fullscreenResolutionX', next)
          }
        />
        <NumberField
          disabled={disabled}
          label={t`Fullscreen resolution height`}
          value={numberValue('fullscreenResolutionY')}
          onChange={(next) =>
            next === undefined
              ? clearField('fullscreenResolutionY')
              : updateField('fullscreenResolutionY', next)
          }
        />
      </Stack>
      <PercentageField
        disabled={disabled}
        kind="zoom"
        value={value?.zoomLevel ?? PERCENT_DEFAULT}
        onChange={(next) => updateField('zoomLevel', next)}
      />
      <PercentageField
        disabled={disabled}
        kind="uiScale"
        value={value?.uiScale ?? PERCENT_DEFAULT}
        onChange={(next) => updateField('uiScale', next)}
      />
      <FormControlLabel
        control={
          <Switch
            disabled={disabled}
            checked={value?.startMuted ?? false}
            onChange={(event) => updateField('startMuted', event.target.checked)}
          />
        }
        label={t`Start muted`}
      />
      <PercentageField
        disabled={disabled}
        kind="music"
        value={value?.musicVolumeLevel ?? PERCENT_DEFAULT}
        onChange={(next) => updateField('musicVolumeLevel', next)}
      />
      <PercentageField
        disabled={disabled}
        kind="sound"
        value={value?.soundVolumeLevel ?? PERCENT_DEFAULT}
        onChange={(next) => updateField('soundVolumeLevel', next)}
      />
    </Stack>
  )
}

export type WindowMode = 'windowed' | 'fullscreen' | 'borderless'

export interface GameSettingsValues {
  windowMode?: WindowMode
  displayIndex?: number
  preferredResolutionX?: number
  preferredResolutionY?: number
  fullscreenResolutionX?: number
  fullscreenResolutionY?: number
  zoomLevel?: number
  uiScale?: number
  startMuted?: boolean
  musicVolumeLevel?: number
  soundVolumeLevel?: number
}
