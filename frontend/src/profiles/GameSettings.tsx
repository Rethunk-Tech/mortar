import { useLingui } from '@lingui/react/macro'
import {
  Button,
  FormControl,
  FormControlLabel,
  InputLabel,
  Menu,
  MenuItem,
  Select,
  Slider,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material'
import { type MouseEvent, type ReactNode, useCallback, useId, useState } from 'react'
import { GameSettings as FetchGameSettings } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { cmpText } from '../mods/cmpText.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { formSettingsFromBackend } from './formSettingsFromBackend.ts'
import { useProfiles } from './store.ts'

const PERCENT_MIN = 0
const PERCENT_MAX = 100
const PERCENT_DEFAULT = 100

interface GameSettingsProps {
  profileId: string
  value: GameSettingsValues | null
  onChange: (value: GameSettingsValues | null) => void
}

interface CopySource {
  id: string
  name: string
}

interface CopyFromProfileProps {
  gameId: string
  profileId: string
  onCopy: (value: GameSettingsValues) => void
}

function CopyFromProfileMenu({ gameId, profileId, onCopy }: CopyFromProfileProps) {
  const { t } = useLingui()
  const profiles = useProfiles((s) => s.profiles)
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const [sources, setSources] = useState<CopySource[]>([])
  const [loading, setLoading] = useState(false)

  const loadSources = useCallback(async () => {
    if (!gameId) {
      setSources([])
      return
    }
    setLoading(true)
    const candidates = profiles.filter((p) => p.id !== profileId)
    const found: CopySource[] = []
    await Promise.all(
      candidates.map(async (profile) => {
        try {
          const raw = await FetchGameSettings(gameId, profile.id)
          if (formSettingsFromBackend(raw) !== null) {
            found.push({ id: profile.id, name: profile.name })
          }
        } catch (error) {
          reportUnexpected(error)
        }
      }),
    )
    found.sort((a, b) => cmpText(a.name, b.name))
    setSources(found)
    setLoading(false)
  }, [gameId, profileId, profiles])

  const openMenu = (event: MouseEvent<HTMLButtonElement>) => {
    setAnchor(event.currentTarget)
    loadSources().catch(reportUnexpected)
  }

  const closeMenu = () => {
    setAnchor(null)
  }

  const pickSource = (sourceId: string) => {
    closeMenu()
    FetchGameSettings(gameId, sourceId)
      .then((raw) => {
        const next = formSettingsFromBackend(raw)
        if (next !== null) {
          onCopy(next)
        }
      })
      .catch(reportUnexpected)
  }

  let menuBody: ReactNode = null
  if (loading) {
    menuBody = <MenuItem disabled={true}>{t`Loading…`}</MenuItem>
  } else if (sources.length === 0) {
    menuBody = <MenuItem disabled={true}>{t`No other profiles with overrides`}</MenuItem>
  } else {
    menuBody = sources.map((source) => (
      <MenuItem key={source.id} onClick={() => pickSource(source.id)}>
        {source.name}
      </MenuItem>
    ))
  }

  return (
    <>
      <Button
        disabled={!gameId}
        size="small"
        aria-haspopup="menu"
        aria-expanded={anchor !== null}
        onClick={openMenu}
      >
        {t`Copy from profile…`}
      </Button>
      <Menu anchorEl={anchor} open={anchor !== null} onClose={closeMenu}>
        {menuBody}
      </Menu>
    </>
  )
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

function parseWindowMode(selected: string): WindowMode | undefined {
  if (selected === '') {
    return undefined
  }
  if (selected === 'windowed' || selected === 'fullscreen' || selected === 'borderless') {
    return selected
  }
  return undefined
}

function WindowModeField({ disabled, value, onChange }: WindowModeFieldProps) {
  const { t } = useLingui()
  const labelId = useId()
  return (
    <FormControl disabled={disabled} fullWidth={true}>
      <InputLabel shrink={true} id={labelId}>{t`Window mode`}</InputLabel>
      <Select
        labelId={labelId}
        displayEmpty={true}
        notched={true}
        label={t`Window mode`}
        value={value ?? ''}
        onChange={(event) => {
          onChange(parseWindowMode(String(event.target.value)))
        }}
      >
        <MenuItem value="">{t`Use default`}</MenuItem>
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
        aria-label={label}
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

interface ProfileGameSettingsFieldsProps {
  disabled: boolean
  value: GameSettingsValues | null
  onChange: (value: GameSettingsValues | null) => void
}

function ProfileGameSettingsFields({ disabled, value, onChange }: ProfileGameSettingsFieldsProps) {
  const { t } = useLingui()

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
    <>
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
        label={t`Display number`}
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
    </>
  )
}

type WindowMode = 'windowed' | 'fullscreen' | 'borderless'

export function GameSettings({ profileId, value, onChange }: GameSettingsProps) {
  const { t } = useLingui()
  const gameId = useProfiles((s) => s.game?.id ?? '')
  const disabled = value === null

  return (
    <Stack spacing={2}>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between' }}>
        <Typography variant="h6">{t`Game settings`}</Typography>
        <CopyFromProfileMenu
          gameId={gameId}
          profileId={profileId}
          onCopy={(next) => onChange(next)}
        />
      </Stack>
      <ProfileGameSettingsFields disabled={disabled} value={value} onChange={onChange} />
    </Stack>
  )
}

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
