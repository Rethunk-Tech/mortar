import { useLingui } from '@lingui/react/macro'
import { Button, Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material'
import { Inbox } from 'lucide-react'
import { useMemo } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  CopyMods,
  UpdateEntry,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useLaunch } from '../launch/store.ts'
import { isLocked } from '../mods/locked.ts'
import { applyWithUndo } from '../mods/menu.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { useToasts } from '../toasts/store.ts'
import { pushUndoToast } from '../toasts/undo.ts'
import { usePending } from '../toasts/usePending.ts'
import { CompareBulkBody } from './CompareBulkBody.tsx'
import type { ComparePair } from './compare.ts'
import { compareProfiles } from './compare.ts'
import { useProfiles } from './store.ts'

const paper = { paper: { sx: { minWidth: 520, maxWidth: 'calc(100vw - 64px)' } } }

export function PickCompareDialog({
  from,
  onPicked,
  onClose,
}: {
  from: Profile | null
  onPicked: (other: Profile) => void
  onClose: () => void
}) {
  const { t } = useLingui()
  const all = useProfiles((s) => s.profiles)
  const profiles = all.filter((p) => p.id !== from?.id)
  return (
    <Dialog open={from !== null} onClose={onClose} slotProps={paper}>
      <DialogTitle>{t`Compare ${from?.name ?? ''} with…`}</DialogTitle>
      <DialogContent>
        {profiles.length === 0 ? (
          <EmptyState
            compact={true}
            icon={<Inbox size={28} />}
            title={t`No other profile of this game.`}
          >
            {t`Create another profile to compare it with this one.`}
          </EmptyState>
        ) : (
          profiles.map((p) => (
            <Button
              key={p.id}
              color="inherit"
              onClick={() => onPicked(p)}
              sx={{ display: 'block', width: 1, justifyContent: 'flex-start' }}
            >
              {p.name}
            </Button>
          ))
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t`Cancel`}</Button>
      </DialogActions>
    </Dialog>
  )
}

export function CompareDialog({
  a,
  b,
  onClose,
}: {
  a: Profile | null
  b: Profile | null
  onClose: () => void
}) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const listed = useProfiles((s) => s.profiles)
  const refresh = useProfiles((s) => s.refresh)
  const aId = a?.id ?? ''
  const bId = b?.id ?? ''
  const open = aId !== '' && bId !== ''
  const profileA = listed.find((p) => p.id === aId) ?? a
  const profileB = listed.find((p) => p.id === bId) ?? b
  const diff = useMemo(() => {
    if (!(profileA && profileB)) {
      return null
    }
    return compareProfiles(profileA, profileB)
  }, [profileA, profileB])
  const [pending, run] = usePending()

  const status = useLaunch((st) => st.status)
  const startingProfile = useLaunch((st) => (st.starting ? st.startingProfile : ''))
  const lockedReason = (profile: Profile) =>
    isLocked(status, profile.id, startingProfile)
      ? t`${profile.name} is in use by the running game. Close the game to change its mods.`
      : ''

  const apply = (to: Profile, change: () => Promise<Profile>, title: string) => {
    run(
      async () => {
        await applyWithUndo(game, to.id, change, (undo) =>
          pushUndoToast(useToasts.getState().push, title, t`Undo`, undo),
        )
        await refresh()
      },
      { errorTitle: t`Could not change ${to.name}` },
    )
  }
  const copy = (from: Profile, to: Profile, uniqueIds: string[]) =>
    apply(
      to,
      () => CopyMods(game, from.id, to.id, uniqueIds),
      t`Copied ${uniqueIds.length} mods to ${to.name}`,
    )
  const match = (from: Profile, to: Profile, rows: ComparePair[]) => {
    const pairs = rows.map((row) => (from.id === profileA?.id ? [row.b, row.a] : [row.a, row.b]))
    apply(
      to,
      async () => {
        let last = to
        for (const [mine, source] of pairs) {
          last = await UpdateEntry(game, to.id, mine?.key ?? '', source?.key ?? '')
        }
        return last
      },
      t`Matched ${rows.length} versions in ${to.name}`,
    )
  }

  const aName = profileA?.name ?? ''
  const bName = profileB?.name ?? ''

  return (
    <Dialog open={open} onClose={pending ? undefined : onClose} slotProps={paper}>
      <DialogTitle>{t`Compare ${aName} and ${bName}`}</DialogTitle>
      <DialogContent>
        {diff && profileA && profileB ? (
          <CompareBulkBody
            diff={diff}
            profileA={profileA}
            profileB={profileB}
            aName={aName}
            bName={bName}
            pending={pending}
            lockedReason={lockedReason}
            onCopy={copy}
            onMatch={match}
          />
        ) : null}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={pending}>{t`Close`}</Button>
      </DialogActions>
    </Dialog>
  )
}
