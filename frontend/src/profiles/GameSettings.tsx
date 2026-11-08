import { useLingui } from '@lingui/react/macro'
import {
  Button,
  FormControlLabel,
  Menu,
  MenuItem,
  Slider,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material'
import { type MouseEvent, memo, type ReactNode, useCallback, useState } from 'react'
import { GameSettings as FetchGameSettings } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { cmpText } from '../mods/cmpText.ts'
import { SelectField } from '../shell/dialog/FormDialog.tsx'
import { MenuAction, MenuNote } from '../shell/MenuAction.tsx'
import { usePopoverAnchor } from '@rethunk/react-load-hooks'
import { reportUnexpected } from '../toasts/report.ts'
import { formSettingsFromBackend } from './formSettingsFromBackend.ts'
import { ProfileDot } from './ProfileDot.tsx'
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
  const { anchor, show, hide } = usePopoverAnchor()
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
    show(event)
    loadSources().catch(reportUnexpected)
  }

  const closeMenu = () => {
    hide()
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
    menuBody = <MenuNote>{t`Loading…`}</MenuNote>
  } else if (sources.length === 0) {
    menuBody = <MenuNote>{t`No other profiles with overrides`}</MenuNote>
  } else {
    menuBody = sources.map((source) => (
      <MenuAction
        key={source.id}
        icon={<ProfileDot profile={profiles.find((p) => p.id === source.id)} />}
        label={source.name}
        onClick={() => pickSource(source.id)}
      />
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

type NumberSettingsField =
  | 'displayIndex'
  | 'preferredResolutionX'
  | 'preferredResolutionY'
  | 'fullscreenResolutionX'
  | 'fullscreenResolutionY'

interface NumberFieldProps {
  disabled: boolean
  label: string
  field: NumberSettingsField
  value: GameSettingsValues | null
  onChange: (value: GameSettingsValues | null) => void
}

function NumberField({ disabled, label, field, value, onChange }: NumberFieldProps) {
  const fieldValue = value?.[field]
  return (
    <TextField
      disabled={disabled}
      fullWidth={true}
      label={label}
      type="number"
      value={typeof fieldValue === 'number' ? fieldValue : ''}
      onChange={(event) => {
        if (event.target.value === '') {
          if (value === null) {
            return
          }
          const next = { ...value }
          delete next[field]
          onChange(Object.keys(next).length === 0 ? null : next)
          return
        }
        onChange({ ...(value ?? {}), [field]: Number(event.target.value) })
      }}
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
  return (
    <SelectField
      disabled={disabled}
      displayEmpty={true}
      label={t`Window mode`}
      value={value ?? ''}
      onChange={(selected) => onChange(parseWindowMode(selected))}
    >
      <MenuItem value="">{t`Use default`}</MenuItem>
      <MenuItem value="windowed">{t`Windowed`}</MenuItem>
      <MenuItem value="fullscreen">{t`Fullscreen`}</MenuItem>
      <MenuItem value="borderless">{t`Borderless`}</MenuItem>
    </SelectField>
  )
}

type PercentageSettingsField = 'zoomLevel' | 'uiScale' | 'musicVolumeLevel' | 'soundVolumeLevel'

interface PercentageFieldProps {
  disabled: boolean
  label: string
  field: PercentageSettingsField
  value: GameSettingsValues | null
  onChange: (value: GameSettingsValues | null) => void
}

function PercentageField({ disabled, label, field, value, onChange }: PercentageFieldProps) {
  const n = value?.[field] ?? PERCENT_DEFAULT
  return (
    <Stack spacing={1}>
      <Typography>{label}</Typography>
      <Slider
        aria-label={label}
        disabled={disabled}
        min={PERCENT_MIN}
        max={PERCENT_MAX}
        value={n}
        valueLabelDisplay="auto"
        onChange={(_, nextValue) => {
          if (typeof nextValue === 'number') {
            onChange({ ...(value ?? {}), [field]: nextValue })
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

// Memoised: a read landing on the row above, or the page's own state, redraws no field while the profile's values
// and the change handler stay the same.
const ProfileGameSettingsFields = memo(function ProfileGameSettingsFieldsView({
  disabled,
  value,
  onChange,
}: ProfileGameSettingsFieldsProps) {
  const { t } = useLingui()
  const zoomLevel = value?.zoomLevel ?? PERCENT_DEFAULT
  const uiScale = value?.uiScale ?? PERCENT_DEFAULT
  const musicVolumeLevel = value?.musicVolumeLevel ?? PERCENT_DEFAULT
  const soundVolumeLevel = value?.soundVolumeLevel ?? PERCENT_DEFAULT

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
        field="displayIndex"
        label={t`Display number`}
        value={value}
        onChange={onChange}
      />
      <Stack direction="row" spacing={2}>
        <NumberField
          disabled={disabled}
          field="preferredResolutionX"
          label={t`Windowed resolution width`}
          value={value}
          onChange={onChange}
        />
        <NumberField
          disabled={disabled}
          field="preferredResolutionY"
          label={t`Windowed resolution height`}
          value={value}
          onChange={onChange}
        />
      </Stack>
      <Stack direction="row" spacing={2}>
        <NumberField
          disabled={disabled}
          field="fullscreenResolutionX"
          label={t`Fullscreen resolution width`}
          value={value}
          onChange={onChange}
        />
        <NumberField
          disabled={disabled}
          field="fullscreenResolutionY"
          label={t`Fullscreen resolution height`}
          value={value}
          onChange={onChange}
        />
      </Stack>
      <PercentageField
        disabled={disabled}
        field="zoomLevel"
        label={t`Zoom (${zoomLevel}%)`}
        value={value}
        onChange={onChange}
      />
      <PercentageField
        disabled={disabled}
        field="uiScale"
        label={t`UI scale (${uiScale}%)`}
        value={value}
        onChange={onChange}
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
        field="musicVolumeLevel"
        label={t`Music volume (${musicVolumeLevel}%)`}
        value={value}
        onChange={onChange}
      />
      <PercentageField
        disabled={disabled}
        field="soundVolumeLevel"
        label={t`Sound volume (${soundVolumeLevel}%)`}
        value={value}
        onChange={onChange}
      />
    </>
  )
})

type WindowMode = 'windowed' | 'fullscreen' | 'borderless'

export function GameSettings({ profileId, value, onChange }: GameSettingsProps) {
  const { t } = useLingui()
  const gameId = useProfiles((s) => s.game?.id ?? '')
  const disabled = value === null

  return (
    <Stack spacing={2}>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between' }}>
        <Typography variant="h6" component="h3">{t`Game settings`}</Typography>
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
