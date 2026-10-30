import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, ButtonBase, Checkbox, Typography } from '@mui/material'
import { useLoaded } from '@rethunk/react-load-hooks'
import { Inbox } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import type {
  DiffSide,
  ModInProfile,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  NeedsToCopy,
  ProfilesWithMod,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { modsLabel } from '../i18n/counts.ts'
import { listNames } from '../i18n/list.ts'
import { userModCount } from '../profiles/count.ts'
import { ProfileMark } from '../profiles/ProfileMark.tsx'
import { useProfiles } from '../profiles/store.ts'
import { AppDialog, DialogFooter } from '../shell/dialog/AppDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { sameValue } from '../shell/sameValue.ts'
import { reportError } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { lockedIn, useLaunchLocks } from './useLocked.ts'

const MARK_PX = 36
const DISABLED_OPACITY = 0.55

function NoOtherProfiles() {
  const { t } = useLingui()
  return (
    <EmptyState compact={true} icon={<Inbox size={28} />} title={t`No other profile of this game`}>
      {t`Create another profile to use this mod there.`}
    </EmptyState>
  )
}

// One profile as a card to tick: its mark, its name, and what it holds or why it cannot change.
function ProfileChoice({
  profile,
  row,
  mode,
  isLockedProfile,
  selected,
  choose,
}: {
  profile: Profile
  row: ModInProfile | undefined
  mode: 'add' | 'remove'
  isLockedProfile: boolean
  selected: boolean
  choose: (id: string) => void
}) {
  const { t } = useLingui()
  const has = row !== undefined
  const disabled = mode === 'remove' ? !has || isLockedProfile : has || isLockedProfile
  const checked = mode === 'remove' ? selected : has || selected
  let status = modsLabel(userModCount(profile))
  if (isLockedProfile) {
    status = t`Stop the game to change mods.`
  } else if (has && mode !== 'remove') {
    status = t`Already has it`
  }
  return (
    <ButtonBase
      role="checkbox"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => choose(profile.id)}
      sx={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'flex-start',
        gap: 1.5,
        p: 1.5,
        textAlign: 'left',
        borderRadius: '8px',
        border: '1px solid',
        borderColor: checked && !disabled ? 'primary.main' : 'var(--mortar-hairline)',
        bgcolor: 'var(--mortar-raised)',
        opacity: disabled ? DISABLED_OPACITY : 1,
      }}
    >
      <ProfileMark profile={profile} size={MARK_PX} />
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography noWrap={true} sx={{ fontWeight: 600 }}>
          {profile.name}
        </Typography>
        <Typography noWrap={true} sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
          {status}
        </Typography>
      </Box>
      <Checkbox
        size="small"
        checked={checked}
        disabled={disabled}
        tabIndex={-1}
        disableRipple={true}
        sx={{ p: 0, pointerEvents: 'none' }}
      />
    </ButtonBase>
  )
}

// What adding the mods to the chosen profiles brings with them, named before the user confirms.
function useNeeds(
  on: boolean,
  { game, from, to, ids }: { game: string; from: string; to: string[]; ids: string[] },
) {
  const toKey = to.join('\n')
  const idsKey = ids.join('\n')
  const { data: needs } = useLoaded<DiffSide[]>(
    on && toKey !== ''
      ? () =>
          NeedsToCopy(game, from, toKey.split('\n'), idsKey.split('\n')).then((list) => list ?? [])
      : null,
    [on, game, from, toKey, idsKey],
    [],
  )
  return needs
}

const toggleSelected = (current: string[], id: string) =>
  current.includes(id) ? current.filter((value) => value !== id) : [...current, id]

function ProfileChoices({
  profiles,
  rows,
  mode,
  selected,
  choose,
  locked,
}: {
  profiles: Profile[]
  rows: ModInProfile[]
  mode: 'add' | 'remove'
  selected: string[]
  choose: (id: string) => void
  locked: (profile: Profile) => boolean
}) {
  return (
    <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 1.5 }}>
      {profiles.map((profile) => {
        const row = rows.find((candidate) => candidate.profileId === profile.id)
        return (
          <ProfileChoice
            key={profile.id}
            profile={profile}
            row={row}
            mode={mode}
            isLockedProfile={locked(profile)}
            selected={selected.includes(profile.id)}
            choose={choose}
          />
        )
      })}
    </Box>
  )
}

// Cancel and the action, with what an add brings along said on the left.
function ChoicesFooter({
  needs,
  busy,
  blocked,
  danger,
  confirmText,
  onCancel,
  onConfirm,
}: {
  needs: DiffSide[]
  busy: boolean
  blocked: boolean
  danger: boolean
  confirmText: string
  onCancel: () => void
  onConfirm: () => void
}) {
  const { t } = useLingui()
  return (
    <DialogFooter
      aside={
        needs.length > 0
          ? t`Also adds what it needs: ${{ names: listNames(needs.map((n) => n.name)) }}`
          : ''
      }
    >
      <Button onClick={onCancel} disabled={busy}>
        {t`Cancel`}
      </Button>
      <Button
        variant="contained"
        color={danger ? 'error' : 'primary'}
        disabled={busy || blocked}
        onClick={onConfirm}
      >
        {confirmText}
      </Button>
    </DialogFooter>
  )
}

// Sets a list only when it differs, so a read that finds what the dialog already shows redraws nothing.
const keep = <T,>(set: (update: (prev: T[]) => T[]) => void, next: T[]) =>
  set((prev) => (sameValue(prev, next) ? prev : next))

const NO_ROWS: ModInProfile[] = []

// The other profiles that have the mods, read as the dialog opens.
function useRows(open: boolean, game: string, currentProfileId: string, which: string | string[]) {
  const { t } = useLingui()
  const key = typeof which === 'string' ? which : which.join('\n')
  const shown = useRef(NO_ROWS)
  const { data: rows } = useLoaded(
    open
      ? () =>
          Promise.all(key.split('\n').map((one) => ProfilesWithMod(game, one))).then((lists) => {
            const next = lists
              .flatMap((list) => list ?? [])
              .filter(
                (row, index, all) =>
                  row.profileId !== currentProfileId &&
                  all.findIndex(
                    (other) => other.profileId === row.profileId && other.key === row.key,
                  ) === index,
              )
            // A read that finds what the dialog already shows redraws nothing.
            return sameValue(shown.current, next) ? shown.current : next
          })
      : null,
    [currentProfileId, game, open, key],
    NO_ROWS,
    reportError(t`Could not read profiles`),
  )
  shown.current = rows
  return rows
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
  // The add brings the mods' requirements along, and the dialog names them.
  withNeeds?: boolean
  onClose: () => void
  onConfirm: (profiles: Profile[], rows: ModInProfile[]) => Promise<void>
}) {
  const { t } = useLingui()
  const listed = useProfiles((s) => s.profiles)
  // Kept stable between renders: the selection effect below depends on it, and a fresh array each render loops.
  const profiles = useMemo(
    () => listed.filter((p) => p.id !== currentProfileId),
    [listed, currentProfileId],
  )
  const rows = useRows(open, game, currentProfileId, ids ?? id)
  const [selected, setSelected] = useState<string[]>([])
  const [pending, run] = usePending()
  const launch = useLaunchLocks()
  const needs = useNeeds(open && withNeeds && mode === 'add', {
    game,
    from: currentProfileId,
    to: selected,
    ids: ids ?? [id],
  })
  // Once the player ticks or unticks a profile, later reads (a game starting, a profile saved) leave the choice alone.
  const touched = useRef<boolean>(false)
  useEffect(() => {
    if (!open) {
      touched.current = false
      return
    }
    if (touched.current) {
      return
    }
    const unavailable = new Set([
      ...(mode === 'add' ? rows.map((row) => row.profileId) : []),
      ...profiles.filter((profile) => lockedIn(launch, profile.id)).map((profile) => profile.id),
    ])
    keep(
      setSelected,
      mode === 'remove'
        ? []
        : profiles.map((profile) => profile.id).filter((profileId) => !unavailable.has(profileId)),
    )
  }, [launch, mode, open, profiles, rows])
  const choose = (profileId: string) => {
    touched.current = true
    setSelected((current) => toggleSelected(current, profileId))
  }
  const locked = (profile: Profile) => lockedIn(launch, profile.id)
  const confirm = () => {
    run(
      async () => {
        const chosen = profiles.filter((profile) => selected.includes(profile.id))
        await onConfirm(
          chosen,
          rows.filter((row) => selected.includes(row.profileId)),
        )
        onClose()
      },
      { errorTitle: t`Could not change mods` },
    )
  }
  const confirmText =
    mode === 'remove'
      ? plural(
          { count: selected.length },
          { one: 'Remove from # profile', other: 'Remove from # profiles' },
        )
      : confirmLabel
  return (
    <AppDialog
      open={open}
      onClose={pending ? () => undefined : onClose}
      title={title}
      subtitle={helper && profiles.length > 0 ? helper : undefined}
      footer={
        <ChoicesFooter
          needs={needs}
          busy={pending}
          blocked={profiles.length === 0 || selected.length === 0}
          danger={mode === 'remove'}
          confirmText={confirmText}
          onCancel={onClose}
          onConfirm={confirm}
        />
      }
    >
      {profiles.length === 0 ? <NoOtherProfiles /> : null}
      <ProfileChoices
        profiles={profiles}
        rows={rows}
        mode={mode}
        selected={selected}
        choose={choose}
        locked={locked}
      />
    </AppDialog>
  )
}
