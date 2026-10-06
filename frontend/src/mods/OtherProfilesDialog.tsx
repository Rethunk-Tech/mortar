import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Checkbox, FormControlLabel, Typography } from '@mui/material'
import { Inbox } from 'lucide-react'
import { useEffect, useState } from 'react'
import type {
  DiffSide,
  ModInProfile,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  NeedsToCopy,
  ProfilesWithMod,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { reportError } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { lockedIn, useLaunchLocks } from './useLocked.ts'

function NoOtherProfiles() {
  const { t } = useLingui()
  return (
    <EmptyState compact={true} icon={<Inbox size={28} />} title={t`No other profile of this game.`}>
      {t`Create another profile to use this mod there.`}
    </EmptyState>
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
          {has && mode !== 'remove' && !isPinned ? ` · ${t`Already has it`}` : ''}
          {isPinned ? ` · ${t`pinned in ${profile.name}`}` : ''}
          {isLockedProfile ? ` · ${t`Stop the game to change mods.`}` : ''}
        </span>
      }
    />
  )
}

// What adding the mods to the chosen profiles brings with them, named before the user confirms.
function useNeeds(
  on: boolean,
  { game, from, to, ids }: { game: string; from: string; to: string[]; ids: string[] },
) {
  const [needs, setNeeds] = useState<DiffSide[]>([])
  const toKey = to.join('\n')
  const idsKey = ids.join('\n')
  useEffect(() => {
    setNeeds([])
    if (!on || toKey === '') {
      return
    }
    let live = true
    NeedsToCopy(game, from, toKey.split('\n'), idsKey.split('\n'))
      .then((list) => live && setNeeds(list ?? []))
      .catch(() => undefined)
    return () => {
      live = false
    }
  }, [on, game, from, toKey, idsKey])
  return needs
}

const toggleSelected = (current: string[], id: string) =>
  current.includes(id) ? current.filter((value) => value !== id) : [...current, id]
const profileHasPinned = (profile: Profile, row: ModInProfile, update?: { oldKey: string }) =>
  Boolean(update && profile.entries?.some((entry) => entry.key === row.key && entry.pinned))

function ProfileChoices({
  profiles,
  rows,
  mode,
  selected,
  choose,
  pinned,
  locked,
}: {
  profiles: Profile[]
  rows: ModInProfile[]
  mode: 'add' | 'remove'
  selected: string[]
  choose: (id: string) => void
  pinned: (profile: Profile, row: ModInProfile) => boolean
  locked: (profile: Profile) => boolean
}) {
  return (
    <>
      {profiles.map((profile) => {
        const row = rows.find((candidate) => candidate.profileId === profile.id)
        const isPinned = row !== undefined && pinned(profile, row)
        return (
          <ProfileChoice
            key={profile.id}
            profile={profile}
            row={row}
            mode={mode}
            isPinned={isPinned}
            isLockedProfile={locked(profile)}
            selected={selected.includes(profile.id)}
            choose={choose}
          />
        )
      })}
    </>
  )
}

export function OtherProfilesDialog({
  open,
  game,
  currentProfileId,
  id,
  ids,
  mode = 'add',
  title,
  helper,
  confirmLabel,
  update,
  withNeeds = false,
  onClose,
  onConfirm,
}: {
  open: boolean
  game: string
  currentProfileId: string
  id: string
  ids?: string[]
  mode?: 'add' | 'remove'
  title: string
  helper?: string
  confirmLabel: string
  update?: { oldKey: string } | undefined
  // The add brings the mods' requirements along, and the dialog names them.
  withNeeds?: boolean
  onClose: () => void
  onConfirm: (profiles: Profile[], pinned: Profile[], rows: ModInProfile[]) => Promise<void>
}) {
  const { t } = useLingui()
  const profiles = useProfiles((s) => s.profiles).filter((p) => p.id !== currentProfileId)
  const [rows, setRows] = useState<ModInProfile[]>([])
  const [selected, setSelected] = useState<string[]>([])
  const [pending, run] = usePending()
  const launch = useLaunchLocks()
  const needs = useNeeds(open && withNeeds && mode === 'add', {
    game,
    from: currentProfileId,
    to: selected,
    ids: ids ?? [id],
  })
  useEffect(() => {
    if (!open) {
      return
    }
    Promise.all((ids ?? [id]).map((one) => ProfilesWithMod(game, one)))
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
      .catch(reportError(t`Could not read other profiles`))
  }, [currentProfileId, game, open, t, id, ids])
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
          return pinned || (profile && lockedIn(launch, profile.id))
        })
        .map((row) => row.profileId),
    )
    setSelected(
      mode === 'remove'
        ? []
        : profiles.map((profile) => profile.id).filter((profileId) => !unavailable.has(profileId)),
    )
  }, [launch, mode, open, profiles, rows, update])
  const choose = (profileId: string) => setSelected((current) => toggleSelected(current, profileId))
  const locked = (profile: Profile) => lockedIn(launch, profile.id)
  const pinned = (profile: Profile, row: ModInProfile) => profileHasPinned(profile, row, update)
  const confirm = () => {
    run(
      async () => {
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
      },
      { errorTitle: t`Could not change mods` },
    )
  }
  const confirmText =
    mode === 'remove'
      ? t`${plural(selected.length, { one: 'Remove from # profile', other: 'Remove from # profiles' })}`
      : confirmLabel
  return (
    <ConfirmDialog
      open={open}
      title={title}
      confirmLabel={confirmText}
      busy={pending}
      confirmDisabled={profiles.length === 0 || selected.length === 0}
      maxWidth={420}
      onCancel={onClose}
      onConfirm={confirm}
    >
      {helper && profiles.length > 0 ? (
        <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
          {helper}
        </Typography>
      ) : null}
      {profiles.length === 0 ? <NoOtherProfiles /> : null}
      <ProfileChoices
        profiles={profiles}
        rows={rows}
        mode={mode}
        selected={selected}
        choose={choose}
        pinned={pinned}
        locked={locked}
      />
      {needs.length > 0 ? (
        <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
          {t`Also adds what it needs: ${needs.map((n) => n.name).join(', ')}`}
        </Typography>
      ) : null}
    </ConfirmDialog>
  )
}
