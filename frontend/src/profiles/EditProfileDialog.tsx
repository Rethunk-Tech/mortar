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
  Typography,
} from '@mui/material'
import { useEffect, useState } from 'react'
import { PickImage } from '../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { applyStagedCover, hasPickedCover, type StagedCover } from '../game/cover.ts'
import { HeroCover } from '../game/HeroCover.tsx'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import {
  clipDescription,
  colorHex,
  MAX_DESCRIPTION,
  PROFILE_COLORS,
  PROFILE_ICONS,
} from './appearance.ts'
import { ProfileMark } from './ProfileMark.tsx'
import { useProfiles } from './store.ts'

const PATH_SEPARATORS = /[\\/]/

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
  return (
    <>
      <Typography sx={{ fontSize: 13, color: 'text.secondary', mb: 1 }}>{t`Colour`}</Typography>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75, mb: 2 }}>
        {PROFILE_COLORS.map((token) => (
          <IconButton
            key={token}
            aria-label={token}
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
        ))}
      </Box>
      <Typography sx={{ fontSize: 13, color: 'text.secondary', mb: 1 }}>{t`Icon`}</Typography>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75, mb: 2 }}>
        {PROFILE_ICONS.map((name) => (
          <IconButton
            key={name}
            aria-label={name}
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
        ))}
      </Box>
    </>
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
  const replace = useProfiles((s) => s.replace)
  const gameId = useProfiles((s) => s.game?.id ?? '')
  const [color, setColor] = useState(profile.color ?? '')
  const [icon, setIcon] = useState(profile.icon ?? '')
  const [description, setDescription] = useState(profile.description ?? '')
  const [launchOptions, setLaunchOptionsField] = useState(profile.launchOptions ?? '')
  const [stagedCover, setStagedCover] = useState<StagedCover>(undefined)
  const [optionsError, setOptionsError] = useState('')
  const [busy, setBusy] = useState(false)
  useEffect(() => {
    if (open) {
      setColor(profile.color ?? '')
      setIcon(profile.icon ?? '')
      setDescription(profile.description ?? '')
      setLaunchOptionsField(profile.launchOptions ?? '')
      setStagedCover(undefined)
      setOptionsError('')
    }
  }, [open, profile.color, profile.description, profile.icon, profile.launchOptions])
  const save = async () => {
    if (busy) {
      return
    }
    setBusy(true)
    setOptionsError('')
    try {
      try {
        await setAppearance(profile.id, color, icon, clipDescription(description))
      } catch (e) {
        reportUnexpected(e)
        return
      }
      try {
        await setLaunchOptions(profile.id, launchOptions)
      } catch (e) {
        setOptionsError(e instanceof Error ? e.message : String(e))
        return
      }
      try {
        const next = await applyStagedCover(gameId, profile.id, stagedCover)
        if (next) {
          replace(next)
        }
      } catch (e) {
        useToasts
          .getState()
          .push({ kind: 'error', title: t`Could not use that image`, body: errorMessage(e) })
        return
      }
      onClose()
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open={open}
      onClose={onClose}
      transitionDuration={0}
      slotProps={{ paper: { sx: { bgcolor: 'rgb(40,40,48)', minWidth: 400 } } }}
    >
      <form
        onSubmit={(e) => {
          e.preventDefault()
          save().catch(reportUnexpected)
        }}
      >
        <DialogTitle>{t`Edit profile`}</DialogTitle>
        <DialogContent>
          <AppearancePickers
            profile={profile}
            color={color}
            icon={icon}
            onColor={setColor}
            onIcon={setIcon}
          />
          <CoverField
            gameId={gameId}
            profile={profile}
            staged={stagedCover}
            onStage={setStagedCover}
          />
          <TextField
            fullWidth={true}
            margin="dense"
            multiline={true}
            minRows={2}
            label={t`Description`}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
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
            onChange={(e) => {
              setLaunchOptionsField(e.target.value)
              setOptionsError('')
            }}
            error={optionsError !== ''}
            helperText={
              optionsError ||
              t`Extra SMAPI arguments for this profile. Mortar sets --mods-path itself.`
            }
            slotProps={{ root: { sx: { userSelect: 'text' } } }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose} sx={{ whiteSpace: 'nowrap' }}>{t`Cancel`}</Button>
          <Button type="submit" variant="contained" disabled={busy} sx={{ whiteSpace: 'nowrap' }}>
            {t`Save`}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  )
}
