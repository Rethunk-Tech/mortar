import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material'
import { useEffect, useState } from 'react'
import { GameSettings as GetGameSettings } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import { PickImage } from '../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { hasPickedCover, type StagedCover } from '../game/cover.ts'
import { HeroCover } from '../game/HeroCover.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { colorHex, MAX_DESCRIPTION, PROFILE_COLORS, PROFILE_ICONS } from './appearance.ts'
import { formSettingsFromBackend } from './formSettingsFromBackend.ts'
import { GameSettings, type GameSettingsValues } from './GameSettings.tsx'
import { LaunchPreview } from './LaunchPreview.tsx'
import { OverridesSection } from './OverrideRows.tsx'
import { foldedOverrides } from './overrideValue.ts'
import { ProfileMark } from './ProfileMark.tsx'
import { saveProfile } from './saveProfile.ts'
import { useProfiles } from './store.ts'

const PATH_SEPARATORS = /[\\/]/

// LaunchError is a rejected launch field: the extra SMAPI arguments, or the prefix and environment saved together.
type LaunchError = { field: 'options' | 'settings'; message: string } | null

function useProfileGameSettings(gameId: string, profileId: string, open: boolean) {
  const [value, setValue] = useState<GameSettingsValues | null>(null)
  const [loaded, setLoaded] = useState(false)
  useEffect(() => {
    let active = true
    if (open) {
      setValue(null)
      setLoaded(false)
      if (gameId) {
        GetGameSettings(gameId, profileId)
          .then((next) => {
            if (active) {
              setValue(formSettingsFromBackend(next))
              setLoaded(true)
            }
          })
          .catch((error) => {
            if (active) {
              reportUnexpected(error)
              setLoaded(true)
            }
          })
      } else {
        setLoaded(true)
      }
    }
    return () => {
      active = false
    }
  }, [gameId, open, profileId])
  return [value, setValue, loaded] as const
}

// The cover choice is staged here and applied by the dialog's Save.
function CoverField({
  gameId,
  profile,
  staged,
  onStage,
}: {
  gameId: string
  profile: Profile
  staged: StagedCover
  onStage: (next: StagedCover) => void
}) {
  const { t } = useLingui()
  const pickedName = typeof staged === 'string' ? (staged.split(PATH_SEPARATORS).pop() ?? '') : ''
  return (
    <>
      <Typography
        sx={{ fontSize: 13, color: 'text.secondary', mb: 1 }}
      >{t`Cover image`}</Typography>
      <Box sx={{ width: 160, height: 90, borderRadius: '6px', overflow: 'hidden', mb: 1 }}>
        {gameId ? <HeroCover game={gameId} profile={profile} /> : null}
      </Box>
      {staged === undefined ? null : (
        <Typography sx={{ fontSize: 12, color: 'text.secondary', mb: 1 }}>
          {staged === null
            ? t`The automatic cover is used after Save.`
            : t`${pickedName} is used after Save.`}
        </Typography>
      )}
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mb: 2 }}>
        <Button
          onClick={() => {
            PickImage(t`Choose image…`)
              .then((path) => {
                if (path) {
                  onStage(path)
                }
              })
              .catch(reportUnexpected)
          }}
          sx={{ whiteSpace: 'nowrap' }}
        >{t`Choose image…`}</Button>
        <Button
          disabled={!hasPickedCover(profile.cover, staged)}
          onClick={() => onStage(null)}
          sx={{ whiteSpace: 'nowrap' }}
        >{t`Use the automatic cover`}</Button>
      </Box>
    </>
  )
}

function AppearancePickers({
  profile,
  color,
  icon,
  onColor,
  onIcon,
}: {
  profile: Profile
  color: string
  icon: string
  onColor: (next: string) => void
  onIcon: (next: string) => void
}) {
  const { t } = useLingui()
  const colorName = (token: string) => {
    switch (token) {
      case 'rose':
        return t`Rose`
      case 'orange':
        return t`Orange`
      case 'gold':
        return t`Gold`
      case 'lime':
        return t`Lime`
      case 'teal':
        return t`Teal`
      case 'sky':
        return t`Sky`
      case 'violet':
        return t`Violet`
      case 'pink':
        return t`Pink`
      default:
        return token
    }
  }
  const iconName = (token: string) => {
    switch (token) {
      case 'sprout':
        return t`Sprout`
      case 'leaf':
        return t`Leaf`
      case 'wheat':
        return t`Wheat`
      case 'fish':
        return t`Fish`
      case 'hammer':
        return t`Hammer`
      case 'pickaxe':
        return t`Pickaxe`
      case 'star':
        return t`Star`
      case 'heart':
        return t`Heart`
      case 'mountain':
        return t`Mountain`
      case 'sun':
        return t`Sun`
      case 'moon':
        return t`Moon`
      case 'sparkles':
        return t`Sparkles`
      default:
        return token
    }
  }
  return (
    <>
      <Typography sx={{ fontSize: 13, color: 'text.secondary', mb: 1 }}>{t`Colour`}</Typography>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75, mb: 2 }}>
        {PROFILE_COLORS.map((token) => (
          <Tooltip key={token} title={colorName(token)}>
            <IconButton
              aria-label={colorName(token)}
              aria-pressed={color === token}
              onClick={() => onColor(color === token ? '' : token)}
              sx={{
                width: 32,
                height: 32,
                bgcolor: colorHex(token),
                outline: color === token ? '2px solid #fff' : '2px solid transparent',
                outlineOffset: 1,
                '&:hover': { bgcolor: colorHex(token) },
              }}
            />
          </Tooltip>
        ))}
      </Box>
      <Typography sx={{ fontSize: 13, color: 'text.secondary', mb: 1 }}>{t`Icon`}</Typography>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75, mb: 2 }}>
        {PROFILE_ICONS.map((name) => (
          <Tooltip key={name} title={iconName(name)}>
            <IconButton
              aria-label={iconName(name)}
              aria-pressed={icon === name}
              onClick={() => onIcon(icon === name ? '' : name)}
              sx={{
                width: 36,
                height: 36,
                borderRadius: '6px',
                bgcolor: icon === name ? 'rgba(255,255,255,0.12)' : 'transparent',
              }}
            >
              <ProfileMark profile={{ ...profile, color, icon: name }} size={28} />
            </IconButton>
          </Tooltip>
        ))}
      </Box>
    </>
  )
}

interface ProfileFieldsProps {
  profile: Profile
  gameId: string
  color: string
  icon: string
  onColor: (value: string) => void
  onIcon: (value: string) => void
  stagedCover: StagedCover
  onStageCover: (value: StagedCover) => void
  description: string
  onDescription: (value: string) => void
  launchOptions: string
  onLaunchOptions: (value: string) => void
  launchPrefix: string
  onLaunchPrefix: (value: string) => void
  launchEnv: string
  onLaunchEnv: (value: string) => void
  launchError: LaunchError
  onLaunchError: (value: LaunchError) => void
  gameSettings: GameSettingsValues | null
  onGameSettings: (value: GameSettingsValues | null) => void
  overrides: Record<string, string>
  onOverrides: (value: Record<string, string>) => void
}

function ProfileFields({
  profile,
  gameId,
  color,
  icon,
  onColor,
  onIcon,
  stagedCover,
  onStageCover,
  description,
  onDescription,
  launchOptions,
  onLaunchOptions,
  launchPrefix,
  onLaunchPrefix,
  launchEnv,
  onLaunchEnv,
  launchError,
  onLaunchError,
  gameSettings,
  onGameSettings,
  overrides,
  onOverrides,
}: ProfileFieldsProps) {
  const { t } = useLingui()
  return (
    <DialogContent>
      <AppearancePickers
        profile={profile}
        color={color}
        icon={icon}
        onColor={onColor}
        onIcon={onIcon}
      />
      <CoverField gameId={gameId} profile={profile} staged={stagedCover} onStage={onStageCover} />
      <TextField
        fullWidth={true}
        margin="dense"
        multiline={true}
        minRows={2}
        label={t`Description`}
        value={description}
        onChange={(event) => onDescription(event.target.value)}
        helperText={`${[...description].length}/${MAX_DESCRIPTION}`}
        slotProps={{
          htmlInput: { maxLength: MAX_DESCRIPTION },
          root: { sx: { userSelect: 'text' } },
        }}
      />
      <TextField
        fullWidth={true}
        margin="dense"
        label={t`Launch options`}
        value={launchOptions}
        onChange={(event) => {
          onLaunchOptions(event.target.value)
          onLaunchError(null)
        }}
        error={launchError?.field === 'options'}
        helperText={
          (launchError?.field === 'options' && launchError.message) ||
          t`Extra SMAPI arguments for this profile. Mortar already sets the mods folder.`
        }
        slotProps={{ root: { sx: { userSelect: 'text' } } }}
      />
      <TextField
        fullWidth={true}
        margin="dense"
        label={t`Launch prefix`}
        value={launchPrefix}
        onChange={(event) => {
          onLaunchPrefix(event.target.value)
          onLaunchError(null)
        }}
        error={launchError?.field === 'settings'}
        helperText={t`Prefix for direct launches only (for example, gamemoderun mangohud). On Windows, prefixes are unavailable.`}
        slotProps={{ root: { sx: { userSelect: 'text' } } }}
      />
      <TextField
        fullWidth={true}
        margin="dense"
        multiline={true}
        minRows={2}
        label={t`Launch environment`}
        value={launchEnv}
        onChange={(event) => {
          onLaunchEnv(event.target.value)
          onLaunchError(null)
        }}
        error={launchError?.field === 'settings'}
        helperText={
          (launchError?.field === 'settings' && launchError.message) ||
          t`One VAR=value per line; applies to direct launches only. Steam launches do not receive these settings.`
        }
        slotProps={{ root: { sx: { userSelect: 'text' } } }}
      />
      <LaunchPreview
        gameId={gameId}
        profileId={profile.id}
        options={launchOptions}
        prefix={launchPrefix}
        env={launchEnv}
      />
      <OverridesSection overrides={overrides} onChange={onOverrides} />
      <GameSettings profileId={profile.id} value={gameSettings} onChange={onGameSettings} />
    </DialogContent>
  )
}

export function EditProfileDialog({
  profile,
  open,
  onClose,
}: {
  profile: Profile
  open: boolean
  onClose: () => void
}) {
  const { t } = useLingui()
  const setAppearance = useProfiles((s) => s.setAppearance)
  const setLaunchOptions = useProfiles((s) => s.setLaunchOptions)
  const setLaunchSettings = useProfiles((s) => s.setLaunchSettings)
  const gameId = useProfiles((s) => s.game?.id ?? '')
  const [color, setColor] = useState(profile.color ?? '')
  const [icon, setIcon] = useState(profile.icon ?? '')
  const [description, setDescription] = useState(profile.description ?? '')
  const [launchOptions, setLaunchOptionsField] = useState(profile.launchOptions ?? '')
  const [launchPrefix, setLaunchPrefix] = useState(profile.launchPrefix ?? '')
  const [launchEnv, setLaunchEnv] = useState(profile.launchEnv ?? '')
  const [overrides, setOverrides] = useState(() => foldedOverrides(profile))
  const [stagedCover, setStagedCover] = useState<StagedCover>(undefined)
  const [gameSettings, setGameSettings, gameSettingsLoaded] = useProfileGameSettings(
    gameId,
    profile.id,
    open,
  )
  const [launchError, setLaunchError] = useState<LaunchError>(null)
  const [busy, setBusy] = useState(false)
  useEffect(() => {
    if (open) {
      setColor(profile.color ?? '')
      setIcon(profile.icon ?? '')
      setDescription(profile.description ?? '')
      setLaunchOptionsField(profile.launchOptions ?? '')
      setLaunchPrefix(profile.launchPrefix ?? '')
      setLaunchEnv(profile.launchEnv ?? '')
      setOverrides(foldedOverrides(profile))
      setStagedCover(undefined)
      setLaunchError(null)
    }
  }, [
    open,
    profile.color,
    profile.description,
    profile.icon,
    profile.launchOptions,
    profile.launchPrefix,
    profile.launchEnv,
    profile,
  ])
  const save = () =>
    saveProfile({
      profile,
      gameId,
      launchOptions,
      launchPrefix,
      launchEnv,
      overrides,
      stagedCover,
      gameSettings,
      setLaunchOptions,
      setLaunchSettings,
      setAppearance,
      color,
      icon,
      description,
      setLaunchError,
      setBusy,
      onClose,
      coverFailure: t`Could not use that image`,
    })
  return (
    <Dialog
      open={open}
      onClose={onClose}
      transitionDuration={0}
      slotProps={{ paper: { sx: { minWidth: 400 } } }}
    >
      <form
        onSubmit={(e) => {
          e.preventDefault()
          save().catch(reportUnexpected)
        }}
      >
        <DialogTitle>{t`Edit profile`}</DialogTitle>
        <ProfileFields
          profile={profile}
          gameId={gameId}
          color={color}
          icon={icon}
          onColor={setColor}
          onIcon={setIcon}
          stagedCover={stagedCover}
          onStageCover={setStagedCover}
          description={description}
          onDescription={setDescription}
          launchOptions={launchOptions}
          onLaunchOptions={setLaunchOptionsField}
          launchPrefix={launchPrefix}
          onLaunchPrefix={setLaunchPrefix}
          launchEnv={launchEnv}
          onLaunchEnv={setLaunchEnv}
          launchError={launchError}
          onLaunchError={setLaunchError}
          gameSettings={gameSettings}
          onGameSettings={setGameSettings}
          overrides={overrides}
          onOverrides={setOverrides}
        />
        <DialogActions>
          <Button onClick={onClose} sx={{ whiteSpace: 'nowrap' }}>{t`Cancel`}</Button>
          <Tooltip title={gameSettingsLoaded ? '' : t`Loading game settings…`}>
            <span>
              <Button
                type="submit"
                variant="contained"
                disabled={busy || !gameSettingsLoaded}
                sx={{ whiteSpace: 'nowrap' }}
              >
                {t`Save`}
              </Button>
            </span>
          </Tooltip>
        </DialogActions>
      </form>
    </Dialog>
  )
}
