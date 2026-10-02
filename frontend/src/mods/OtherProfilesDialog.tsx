import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  Typography,
} from '@mui/material'
import { useEffect, useState } from 'react'
import type {
  ModInProfile,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { ProfilesWithMod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useLaunch } from '../launch/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { isLocked } from './locked.ts'
import { selectableProfileIds } from './otherProfiles.ts'

export function OtherProfilesDialog({
  open,
  game,
  currentProfileId,
  uniqueId,
  title,
  confirmLabel,
  update,
  onClose,
  onConfirm,
}: {
  open: boolean
  game: string
  currentProfileId: string
  uniqueId: string
  title: string
  confirmLabel: string
  update?: { oldKey: string } | undefined
  onClose: () => void
  onConfirm: (profiles: Profile[], pinned: Profile[]) => Promise<void>
}) {
  const { t } = useLingui()
  const profiles = useProfiles((s) => s.profiles).filter((p) => p.id !== currentProfileId)
  const [rows, setRows] = useState<ModInProfile[]>([])
  const [selected, setSelected] = useState<string[]>([])
  const [pending, setPending] = useState(false)
  const launch = useLaunch((s) => ({
    status: s.status,
    starting: s.starting,
    startingProfile: s.startingProfile,
  }))

  useEffect(() => {
    if (!open) {
      return
    }
    ProfilesWithMod(game, uniqueId)
      .then((next) => setRows((next ?? []).filter((row) => row.profileId !== currentProfileId)))
      .catch((e) =>
        useToasts
          .getState()
          .push({ kind: 'error', title: t`Could not read other profiles`, body: errorMessage(e) }),
      )
  }, [currentProfileId, game, open, t, uniqueId])

  useEffect(() => {
    if (!open) {
      return
    }
    const unavailable = new Set(
      rows
        .filter((row) => {
          const profile = profiles.find((p) => p.id === row.profileId)
          const pinned =
            update && profile?.entries?.some((entry) => entry.key === row.key && entry.pinned)
          return (
            pinned ||
            (profile &&
              isLocked(launch.status, profile.id, launch.starting ? launch.startingProfile : ''))
          )
        })
        .map((row) => row.profileId),
    )
    setSelected(
      selectableProfileIds(
        rows.map((row) => row.profileId),
        unavailable,
      ),
    )
  }, [launch.starting, launch.startingProfile, launch.status, open, profiles, rows, update])

  const choose = (id: string) =>
    setSelected((current) =>
      current.includes(id) ? current.filter((value) => value !== id) : [...current, id],
    )
  const locked = (profile: Profile) =>
    isLocked(launch.status, profile.id, launch.starting ? launch.startingProfile : '')
  const pinned = (profile: Profile, row: ModInProfile) =>
    Boolean(update && profile.entries?.some((entry) => entry.key === row.key && entry.pinned))
  const confirm = async () => {
    setPending(true)
    try {
      const chosen = profiles.filter((profile) => selected.includes(profile.id))
      const skipped = rows
        .filter((row) => {
          const profile = profiles.find((p) => p.id === row.profileId)
          return profile !== undefined && pinned(profile, row)
        })
        .flatMap((row) => profiles.filter((profile) => profile.id === row.profileId))
      await onConfirm(chosen, skipped)
      onClose()
    } catch (e) {
      useToasts
        .getState()
        .push({ kind: 'error', title: t`Could not change mods`, body: errorMessage(e) })
    } finally {
      setPending(false)
    }
  }

  return (
    <Dialog open={open} onClose={pending ? undefined : onClose} transitionDuration={0}>
      <DialogTitle>{title}</DialogTitle>
      <DialogContent sx={{ minWidth: 420 }}>
        {profiles.length === 0 ? (
          <Typography>{t`No other profile of this game.`}</Typography>
        ) : null}
        {profiles.map((profile) => {
          const row = rows.find((candidate) => candidate.profileId === profile.id)
          const has = row !== undefined
          const isPinned = row !== undefined && pinned(profile, row)
          const isLockedProfile = locked(profile)
          const disabled = has || isPinned || isLockedProfile
          return (
            <FormControlLabel
              key={profile.id}
              control={
                <Checkbox
                  checked={has || selected.includes(profile.id)}
                  disabled={disabled}
                  onChange={() => choose(profile.id)}
                />
              }
              label={
                <span>
                  {profile.name}
                  {has && !isPinned ? ` — ${t`Already has it`}` : ''}
                  {isPinned ? ` — ${t`pinned in ${profile.name}`}` : ''}
                  {isLockedProfile ? ` — ${t`Stop the game to change mods.`}` : ''}
                </span>
              }
            />
          )
        })}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={pending}>{t`Cancel`}</Button>
        <Button
          onClick={() => confirm().catch(() => undefined)}
          disabled={pending || selected.length === 0}
          variant="contained"
        >
          {confirmLabel}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
