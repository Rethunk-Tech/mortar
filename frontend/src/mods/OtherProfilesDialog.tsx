import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
} from '@mui/material'
import { Inbox } from 'lucide-react'
import { useEffect, useState } from 'react'
import type {
  ModInProfile,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { ProfilesWithMod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useLaunch } from '../launch/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { isLocked } from './locked.ts'
import { selectableProfileIds } from './otherProfiles.ts'

function NoOtherProfiles() {
  const { t } = useLingui()
  return (
    <EmptyState compact={true} icon={<Inbox size={28} />} title={t`No other profile of this game.`}>
      {t`Create another profile to use this mod there.`}
    </EmptyState>
  )
}

function DialogFooter({
  pending,
  selected,
  confirmText,
  onClose,
  onConfirm,
}: {
  pending: boolean
  selected: number
  confirmText: string
  onClose: () => void
  onConfirm: () => void
}) {
  const { t } = useLingui()
  return (
    <DialogActions>
      <Button onClick={onClose} disabled={pending}>
        {t`Cancel`}
      </Button>
      <Button onClick={onConfirm} disabled={pending || selected === 0} variant="contained">
        {confirmText}
      </Button>
    </DialogActions>
  )
}

function ProfileChoice({
  profile,
  row,
  mode,
  isPinned,
  isLockedProfile,
  selected,
  choose,
}: {
  profile: Profile
  row: ModInProfile | undefined
  mode: 'add' | 'remove'
  isPinned: boolean
  isLockedProfile: boolean
  selected: boolean
  choose: (id: string) => void
}) {
  const { t } = useLingui()
  const has = row !== undefined
  const disabled =
    mode === 'remove' ? !has || isPinned || isLockedProfile : has || isPinned || isLockedProfile
  return (
    <FormControlLabel
      control={
        <Checkbox
          checked={mode === 'remove' ? selected : has || selected}
          disabled={disabled}
          onChange={() => choose(profile.id)}
        />
      }
      label={
        <span>
          {profile.name}
          {has && mode !== 'remove' && !isPinned ? ` — ${t`Already has it`}` : ''}
          {isPinned ? ` — ${t`pinned in ${profile.name}`}` : ''}
          {isLockedProfile ? ` — ${t`Stop the game to change mods.`}` : ''}
        </span>
      }
    />
  )
}

const toggleSelected = (current: string[], id: string) =>
  current.includes(id) ? current.filter((value) => value !== id) : [...current, id]
const profileIsLocked = (
  status: Parameters<typeof isLocked>[0],
  starting: boolean,
  startingProfile: string,
  profile: Profile,
) => isLocked(status, profile.id, starting ? startingProfile : '')
const profileHasPinned = (profile: Profile, row: ModInProfile, update?: { oldKey: string }) =>
  Boolean(update && profile.entries?.some((entry) => entry.key === row.key && entry.pinned))

export function OtherProfilesDialog({
  open,
  game,
  currentProfileId,
  uniqueId,
  uniqueIds,
  mode = 'add',
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
  uniqueIds?: string[]
  mode?: 'add' | 'remove'
  title: string
  confirmLabel: string
  update?: { oldKey: string } | undefined
  onClose: () => void
  onConfirm: (profiles: Profile[], pinned: Profile[], rows: ModInProfile[]) => Promise<void>
}) {
  const { t } = useLingui()
  const profiles = useProfiles((s) => s.profiles).filter((p) => p.id !== currentProfileId)
  const [rows, setRows] = useState<ModInProfile[]>([])
  const [selected, setSelected] = useState<string[]>([])
  const [pending, setPending] = useState(false)
  const launchStatus = useLaunch((s) => s.status)
  const launchStarting = useLaunch((s) => s.starting)
  const launchStartingProfile = useLaunch((s) => s.startingProfile)
  useEffect(() => {
    if (!open) {
      return
    }
    Promise.all((uniqueIds ?? [uniqueId]).map((id) => ProfilesWithMod(game, id)))
      .then((lists) =>
        setRows(
          lists
            .flatMap((next) => next ?? [])
            .filter(
              (row, index, all) =>
                row.profileId !== currentProfileId &&
                all.findIndex(
                  (other) => other.profileId === row.profileId && other.key === row.key,
                ) === index,
            ),
        ),
      )
      .catch((e) =>
        useToasts
          .getState()
          .push({ kind: 'error', title: t`Could not read other profiles`, body: errorMessage(e) }),
      )
  }, [currentProfileId, game, open, t, uniqueId, uniqueIds])
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
              isLocked(launchStatus, profile.id, launchStarting ? launchStartingProfile : ''))
          )
        })
        .map((row) => row.profileId),
    )
    setSelected(
      mode === 'remove'
        ? []
        : selectableProfileIds(
            profiles.map((profile) => profile.id),
            unavailable,
          ),
    )
  }, [launchStarting, launchStartingProfile, launchStatus, mode, open, profiles, rows, update])
  const choose = (id: string) => setSelected((current) => toggleSelected(current, id))
  const locked = (profile: Profile) =>
    profileIsLocked(launchStatus, launchStarting, launchStartingProfile, profile)
  const pinned = (profile: Profile, row: ModInProfile) => profileHasPinned(profile, row, update)
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
      await onConfirm(
        chosen,
        skipped,
        rows.filter((row) => selected.includes(row.profileId)),
      )
      onClose()
    } catch (e) {
      useToasts
        .getState()
        .push({ kind: 'error', title: t`Could not change mods`, body: errorMessage(e) })
    } finally {
      setPending(false)
    }
  }
  const confirmText = mode === 'remove' ? t`Remove from ${selected.length} profiles` : confirmLabel
  return (
    <Dialog open={open} onClose={pending ? undefined : onClose} transitionDuration={0}>
      <DialogTitle>{title}</DialogTitle>
      <DialogContent sx={{ minWidth: 420, maxWidth: 'calc(100vw - 64px)' }}>
        {profiles.length === 0 ? <NoOtherProfiles /> : null}
        {profiles.map((profile) => {
          const row = rows.find((candidate) => candidate.profileId === profile.id)
          const isPinned = row !== undefined && pinned(profile, row)
          const isLockedProfile = locked(profile)
          return (
            <ProfileChoice
              key={profile.id}
              profile={profile}
              row={row}
              mode={mode}
              isPinned={isPinned}
              isLockedProfile={isLockedProfile}
              selected={selected.includes(profile.id)}
              choose={choose}
            />
          )
        })}
      </DialogContent>
      <DialogFooter
        pending={pending}
        selected={profiles.length === 0 ? 0 : selected.length}
        confirmText={confirmText}
        onClose={onClose}
        onConfirm={() => confirm().catch(() => undefined)}
      />
    </Dialog>
  )
}
