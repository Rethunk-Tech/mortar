import { useLingui } from '@lingui/react/macro'
import { Box, Button, Tab, Tabs, TextField, Typography } from '@mui/material'
import { type ReactNode, useEffect, useId, useRef, useState } from 'react'
import { PickImage } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { hasPickedCover, type StagedCover } from '../game/cover.ts'
import { HeroCover } from '../game/HeroCover.tsx'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { Field, FormDialog, FormSection } from '../shell/dialog/FormDialog.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { AppearancePickers } from './AppearancePickers.tsx'
import { MAX_NAME } from './appearance.ts'
import { LaunchOptionsBlock } from './LaunchOptionsBlock.tsx'
import { LaunchPresetsBlock } from './LaunchPresetsBlock.tsx'
import { LoaderPicker } from './LoaderPicker.tsx'
import { SeparateSavesRow } from './SeparateSavesRow.tsx'
import { ServerSettings } from './ServerSettings.tsx'
import { saveProfile } from './saveProfile.ts'
import { useProfiles } from './store.ts'
import { useNameField } from './useNameField.ts'

const PATH_SEPARATORS = /[\\/]/

type FieldsTab = 'appearance' | 'launch'

// LaunchError is a rejected launch field: the extra SMAPI arguments, or the prefix and environment saved together.
type LaunchError = { field: 'options' | 'settings'; message: string } | null

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
    <Field label={t`Cover image`}>
      <Box sx={{ width: 160, height: 90, borderRadius: '6px', overflow: 'hidden', mb: 1 }}>
        {gameId ? <HeroCover game={gameId} profile={profile} /> : null}
      </Box>
      {staged === undefined ? null : (
        <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary', mb: 1 }}>
          {staged === null
            ? t`The automatic cover is used after Save.`
            : t`${{ name: pickedName }} is used after Save.`}
        </Typography>
      )}
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: space.gap, ml: -1 }}>
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
    </Field>
  )
}

function FieldsTabs({
  ids,
  tab,
  onTab,
}: {
  ids: string
  tab: FieldsTab
  onTab: (tab: FieldsTab) => void
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
        mx: -1.75,
        '& .MuiTabs-indicator': { height: 2 },
        '& .MuiTab-root': {
          minHeight: 44,
          minWidth: 0,
          px: '14px',
          fontSize: 'medium.fontSize',
          fontWeight: 400,
          color: 'text.secondary',
          '&.Mui-selected': { color: 'var(--mortar-ink)', fontWeight: 600 },
        },
      }}
    >
      <Tab {...tabProps('appearance')} label={t`Appearance`} />
      <Tab {...tabProps('launch')} label={t`Launch`} />
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
  launchOptions: string
  onLaunchOptions: (value: string) => void
  launchPrefix: string
  onLaunchPrefix: (value: string) => void
  launchEnv: string
  onLaunchEnv: (value: string) => void
  launchError: LaunchError
  onLaunchError: (value: LaunchError) => void
  ids: string
  tab: FieldsTab
}

// Only the open panel is drawn; what a field holds lives in the dialog, so switching tabs loses nothing typed.
const panelSx = {
  // Room inside the scroll box for a slider's thumb at its far end.
  overflowX: 'hidden',
  px: 1,
  mx: -1,
} as const

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
  launchOptions,
  onLaunchOptions,
  launchPrefix,
  onLaunchPrefix,
  launchEnv,
  onLaunchEnv,
  launchError,
  onLaunchError,
  ids,
  tab,
}: ProfileFieldsProps) {
  const { t } = useLingui()
  const nameId = useId()
  const panel = (id: FieldsTab, children: ReactNode) =>
    tab === id ? (
      <Box
        role="tabpanel"
        id={`${ids}-panel-${id}`}
        aria-labelledby={`${ids}-tab-${id}`}
        sx={{ ...panelSx, display: 'flex', flexDirection: 'column', gap: 3 }}
      >
        {children}
      </Box>
    ) : null
  return (
    <Box>
      {panel(
        'appearance',
        <>
          <Field label={t`Name`} htmlFor={nameId} error={nameError || undefined}>
            <TextField
              id={nameId}
              fullWidth={true}
              size="small"
              autoFocus={focusName}
              value={name}
              onChange={(event) => onName(event.target.value)}
              onFocus={(event) => event.target.select()}
              error={nameError !== ''}
              slotProps={{
                htmlInput: { maxLength: MAX_NAME },
                root: { sx: { userSelect: 'text' } },
              }}
            />
          </Field>
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
          <FormSection title={t`Command line`}>
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
          </FormSection>
          <ServerSettings gameId={gameId} profileId={profile.id} />
          <LaunchPresetsBlock
            gameId={gameId}
            profileId={profile.id}
            launchOptions={launchOptions}
            launchPrefix={launchPrefix}
            launchEnv={launchEnv}
          />
        </>,
      )}
    </Box>
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

// The open tab; a launch field Save rejected is shown, wherever the user was.
function useFieldsTab(initialTab: FieldsTab, launchError: LaunchError) {
  const ids = useId()
  const [tab, setTab] = useState<FieldsTab>(initialTab)
  useEffect(() => {
    if (launchError) {
      setTab('launch')
    }
  }, [launchError])
  const bar = <FieldsTabs ids={ids} tab={tab} onTab={setTab} />
  return { ids, tab, bar }
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
  const [launchOptions, setLaunchOptionsField] = useState(profile.launchOptions ?? '')
  const [launchPrefix, setLaunchPrefix] = useState(profile.launchPrefix ?? '')
  const [launchEnv, setLaunchEnv] = useState(profile.launchEnv ?? '')
  const [stagedCover, setStagedCover] = useState<StagedCover>(undefined)
  const [launchError, setLaunchError] = useState<LaunchError>(null)
  const { ids, tab, bar } = useFieldsTab(initialTab, launchError)
  const [busy, setBusy] = useState(false)
  const [touched, setTouched] = useState(false)
  const edited = <A,>(set: (value: A) => void) => touching(setTouched, set)
  // Fields fill once per opening: each save step hands back a new profile, and refilling then would wipe text a later
  // step rejected.
  const filledFor = useRef('')
  useEffect(() => {
    if (!open) {
      filledFor.current = ''
      return
    }
    if (filledFor.current !== profile.id) {
      filledFor.current = profile.id
      setTouched(false)
      setColor(profile.color ?? '')
      setIcon(profile.icon ?? '')
      setLaunchOptionsField(profile.launchOptions ?? '')
      setLaunchPrefix(profile.launchPrefix ?? '')
      setLaunchEnv(profile.launchEnv ?? '')
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
      stagedCover,
      setLaunchOptions,
      setLaunchSettings,
      setAppearance,
      color,
      icon,
      setLaunchError,
      setBusy,
      onClose,
      coverFailure: t`Could not use that image`,
    })
  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title={t`Edit profile`}
      size="lg"
      submitLabel={t`Save`}
      dirty={touched}
      busy={busy}
      onSubmit={() => save().catch(reportUnexpected)}
      tabs={bar}
    >
      <ProfileFields
        ids={ids}
        tab={tab}
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
        launchOptions={launchOptions}
        onLaunchOptions={edited(setLaunchOptionsField)}
        launchPrefix={launchPrefix}
        onLaunchPrefix={edited(setLaunchPrefix)}
        launchEnv={launchEnv}
        onLaunchEnv={edited(setLaunchEnv)}
        launchError={launchError}
        onLaunchError={setLaunchError}
      />
    </FormDialog>
  )
}
