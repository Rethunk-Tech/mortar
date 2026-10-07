import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Tab,
  Tabs,
  TextField,
  Typography,
} from '@mui/material'
import { type ReactNode, useEffect, useId, useState } from 'react'
import { GameSettings as GetGameSettings } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { PickImage } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { hasPickedCover, type StagedCover } from '../game/cover.ts'
import { HeroCover } from '../game/HeroCover.tsx'
import { useGameInfo } from '../games/info.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { useDiscardGuard } from '../shell/useDiscardGuard.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { AppearancePickers } from './AppearancePickers.tsx'
import { MAX_DESCRIPTION, MAX_NAME } from './appearance.ts'
import { formSettingsFromBackend } from './formSettingsFromBackend.ts'
import { GameSettings, type GameSettingsValues } from './GameSettings.tsx'
import { LaunchOptionsBlock } from './LaunchOptionsBlock.tsx'
import { LaunchPresetsBlock } from './LaunchPresetsBlock.tsx'
import { LoaderLaunchSettings } from './LoaderLaunchSettings.tsx'
import { LoaderPicker } from './LoaderPicker.tsx'
import { OverridesSection } from './OverrideRows.tsx'
import { foldedOverrides } from './overrideValue.ts'
import { SeparateSavesRow } from './SeparateSavesRow.tsx'
import { saveProfile } from './saveProfile.ts'
import { useProfiles } from './store.ts'
import { useNameField } from './useNameField.ts'

const PATH_SEPARATORS = /[\\/]/

type FieldsTab = 'appearance' | 'launch' | 'overrides' | 'game'

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
  const picked = hasPickedCover(profile.cover, staged)
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
        >{t`Choose image…`}</Button>
        <DisabledReason title={t`The automatic cover is already in use.`} disabled={!picked}>
          <Button disabled={!picked} onClick={() => onStage(null)}>{t`Use default`}</Button>
        </DisabledReason>
      </Box>
    </>
  )
}

function FieldsTabs({
  ids,
  tab,
  onTab,
  startupSettings,
}: {
  ids: string
  tab: FieldsTab
  onTab: (tab: FieldsTab) => void
  startupSettings: boolean
}) {
  const { t } = useLingui()
  const tabProps = (id: FieldsTab) => ({
    value: id,
    id: `${ids}-tab-${id}`,
    'aria-controls': `${ids}-panel-${id}`,
  })
  return (
    <Tabs
      value={tab}
      onChange={(_, next: FieldsTab) => onTab(next)}
      aria-label={t`Profile settings`}
      sx={{
        minHeight: 44,
        px: 1.5,
        borderBottom: '1px solid var(--mortar-hairline)',
        '& .MuiTabs-indicator': { height: 2 },
        '& .MuiTab-root': {
          minHeight: 44,
          minWidth: 0,
          px: '14px',
          fontSize: 14,
          fontWeight: 400,
          color: 'text.secondary',
          '&.Mui-selected': { color: 'var(--mortar-ink)', fontWeight: 600 },
        },
      }}
    >
      <Tab {...tabProps('appearance')} label={t`Appearance`} />
      <Tab {...tabProps('launch')} label={t`Launch`} />
      <Tab {...tabProps('overrides')} label={t`Overrides`} />
      {startupSettings ? <Tab {...tabProps('game')} label={t`Game settings`} /> : null}
    </Tabs>
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
  name: string
  onName: (value: string) => void
  nameError: string
  focusName: boolean
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
  initialTab: FieldsTab
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
  name,
  onName,
  nameError,
  focusName,
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
  initialTab,
}: ProfileFieldsProps) {
  const { t } = useLingui()
  const ids = useId()
  // Only a game with a startup preferences file applies them; elsewhere the fields would change nothing.
  const startupSettings = useGameInfo(gameId)?.startupSettings === true
  const [tab, setTab] = useState<FieldsTab>(initialTab)
  // A launch field Save rejected is shown, wherever the user was.
  useEffect(() => {
    if (launchError) {
      setTab('launch')
    }
  }, [launchError])
  // Every panel stays mounted and shares one grid cell, so a field keeps what was typed in it and the dialog keeps
  // the height of its tallest tab while another is open.
  const panel = (id: FieldsTab, children: ReactNode) => (
    <Box
      role="tabpanel"
      id={`${ids}-panel-${id}`}
      aria-labelledby={`${ids}-tab-${id}`}
      sx={{ gridArea: '1 / 1', visibility: tab === id ? 'visible' : 'hidden' }}
    >
      {children}
    </Box>
  )
  return (
    <>
      <FieldsTabs ids={ids} tab={tab} onTab={setTab} startupSettings={startupSettings} />
      <DialogContent sx={{ pt: 2.5, display: 'grid', alignContent: 'start' }}>
        {panel(
          'appearance',
          <>
            <TextField
              fullWidth={true}
              margin="dense"
              autoFocus={focusName}
              label={t`Name`}
              value={name}
              onChange={(event) => onName(event.target.value)}
              onFocus={(event) => event.target.select()}
              error={nameError !== ''}
              helperText={nameError || undefined}
              slotProps={{
                htmlInput: { maxLength: MAX_NAME },
                root: { sx: { userSelect: 'text' } },
              }}
            />
            <AppearancePickers
              profile={profile}
              color={color}
              icon={icon}
              onColor={onColor}
              onIcon={onIcon}
            />
            <CoverField
              gameId={gameId}
              profile={profile}
              staged={stagedCover}
              onStage={onStageCover}
            />
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
          </>,
        )}
        {panel(
          'launch',
          <>
            <LoaderPicker gameId={gameId} profileId={profile.id} loader={profile.loader ?? ''} />
            <SeparateSavesRow
              gameId={gameId}
              profileId={profile.id}
              on={profile.separateSaves ?? false}
            />
            <LaunchOptionsBlock
              gameId={gameId}
              profileId={profile.id}
              launchOptions={launchOptions}
              onLaunchOptions={onLaunchOptions}
              launchPrefix={launchPrefix}
              onLaunchPrefix={onLaunchPrefix}
              launchEnv={launchEnv}
              onLaunchEnv={onLaunchEnv}
              launchError={launchError}
              onLaunchError={onLaunchError}
            />
            <LoaderLaunchSettings
              key={profile.loader ?? ''}
              gameId={gameId}
              profileId={profile.id}
              loader={profile.loader ?? ''}
            />
            <LaunchPresetsBlock
              gameId={gameId}
              profileId={profile.id}
              launchOptions={launchOptions}
              launchPrefix={launchPrefix}
              launchEnv={launchEnv}
            />
          </>,
        )}
        {panel('overrides', <OverridesSection overrides={overrides} onChange={onOverrides} />)}
        {startupSettings
          ? panel(
              'game',
              <GameSettings
                profileId={profile.id}
                value={gameSettings}
                onChange={onGameSettings}
              />,
            )
          : null}
      </DialogContent>
    </>
  )
}

// touching marks the form as edited before it applies a field's change.
function touching<A>(setTouched: (touched: boolean) => void, set: (value: A) => void) {
  return (value: A) => {
    setTouched(true)
    set(value)
  }
}

interface EditProfileDialogProps {
  profile: Profile
  open: boolean
  onClose: () => void
  // The tab Edit profile opens on.
  initialTab?: FieldsTab
  // Puts the cursor in the Name field, for a rename.
  focusName?: boolean
}

export function EditProfileDialog(props: EditProfileDialogProps) {
  const { profile, open, onClose, initialTab = 'appearance', focusName = false } = props
  const { t } = useLingui()
  const setAppearance = useProfiles((s) => s.setAppearance)
  const setLaunchOptions = useProfiles((s) => s.setLaunchOptions)
  const setLaunchSettings = useProfiles((s) => s.setLaunchSettings)
  const gameId = useProfiles((s) => s.game?.id ?? '')
  const [color, setColor] = useState(profile.color ?? '')
  const [icon, setIcon] = useState(profile.icon ?? '')
  const { name, setName, nameError, setNameError } = useNameField(profile.name, open)
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
  const [touched, setTouched] = useState(false)
  const guard = useDiscardGuard(touched, onClose)
  const edited = <A,>(set: (value: A) => void) => touching(setTouched, set)
  useEffect(() => {
    if (open) {
      setTouched(false)
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
  }, [open, profile])
  const save = () =>
    saveProfile({
      profile,
      name,
      setNameError,
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
    <Dialog open={open} onClose={guard.request} fullWidth={true}>
      {/* The dialog is as tall as its tallest tab (capped by the window), so only the fields scroll, under fixed tabs and above a fixed footer. */}
      <Box
        component="form"
        onSubmit={(e) => {
          e.preventDefault()
          save().catch(reportUnexpected)
        }}
        sx={{ display: 'flex', flexDirection: 'column', minHeight: 0, flex: '1 1 auto' }}
      >
        <DialogTitle>{t`Edit profile`}</DialogTitle>
        <ProfileFields
          initialTab={initialTab}
          profile={profile}
          gameId={gameId}
          color={color}
          icon={icon}
          onColor={edited(setColor)}
          onIcon={edited(setIcon)}
          stagedCover={stagedCover}
          onStageCover={edited(setStagedCover)}
          name={name}
          onName={edited(setName)}
          nameError={nameError}
          focusName={focusName}
          description={description}
          onDescription={edited(setDescription)}
          launchOptions={launchOptions}
          onLaunchOptions={edited(setLaunchOptionsField)}
          launchPrefix={launchPrefix}
          onLaunchPrefix={edited(setLaunchPrefix)}
          launchEnv={launchEnv}
          onLaunchEnv={edited(setLaunchEnv)}
          launchError={launchError}
          onLaunchError={setLaunchError}
          gameSettings={gameSettings}
          onGameSettings={edited(setGameSettings)}
          overrides={overrides}
          onOverrides={edited(setOverrides)}
        />
        <DialogActions>
          <Button onClick={guard.request}>{t`Cancel`}</Button>
          <DisabledReason title={t`Loading game settings…`} disabled={!gameSettingsLoaded}>
            <Button type="submit" variant="contained" disabled={busy || !gameSettingsLoaded}>
              {t`Save`}
            </Button>
          </DisabledReason>
        </DialogActions>
      </Box>
      {guard.dialog}
    </Dialog>
  )
}
