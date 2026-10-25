import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { ArrowLeft, ArrowRight, GitCompare, ListChecks, Users } from 'lucide-react'
import { type ReactNode, useMemo, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  CopyMods,
  UpdateEntries,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { FarmDialog } from '../game/FarmMenuItem.tsx'
import { hasModList } from '../game/modList.ts'
import { applyWithUndo } from '../mods/menu.ts'
import { lockedIn, useLaunchLocks } from '../mods/useLocked.ts'
import { openImport } from '../share/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { AppDialog, DialogFooter, ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { space } from '../theme/density.ts'
import { useToasts } from '../toasts/store.ts'
import { pushUndoToast } from '../toasts/undo.ts'
import { usePending } from '../toasts/usePending.ts'
import { CompareGroupView, CompareHeader } from './CompareTable.tsx'
import { CompareTop } from './CompareTop.tsx'
import type { CompareRow, CompareView, HideableKind } from './compare.ts'
import { applyPlan, compareProfiles, compareView } from './compare.ts'
import { useGroupTitle } from './compareTitle.ts'
import { useProfiles } from './store.ts'

// Rows that exist only in A are a long tail, so they start unchecked; identical rows have nothing to apply.
const checkedByDefault = (row: CompareRow) => row.kind !== 'onlyA'

function useMatch(profileA: Profile, profileB: Profile, selected: CompareRow[]) {
  const { t } = useLingui()
  const gameId = useProfiles((s) => s.game?.id ?? '')
  const refresh = useProfiles((s) => s.refresh)
  const [pending, run] = usePending()
  const launch = useLaunchLocks()
  const lockedReason = (profile: Profile) =>
    lockedIn(launch, profile.id)
      ? t`${{ name: profile.name }} is in use by the running game. Close the game to change its mods.`
      : ''
  const apply = (to: Profile, change: () => Promise<Profile>, title: string) =>
    applyWithUndo(gameId, to.id, change, (undo) =>
      pushUndoToast(useToasts.getState().push, title, {
        label: t`Undo`,
        run: undo,
        profileId: to.id,
      }),
    )
  const matchTo = (toB: boolean) => {
    const [from, to] = toB ? [profileA, profileB] : [profileB, profileA]
    const plan = applyPlan(selected, toB)
    run(
      async () => {
        if (plan.copy.length > 0) {
          await apply(
            to,
            () => CopyMods(gameId, from.id, to.id, plan.copy),
            plural(
              { count: plan.copy.length },
              {
                one: `Copied # mod to ${{ name: to.name }}`,
                other: `Copied # mods to ${{ name: to.name }}`,
              },
            ),
          )
        }
        if (plan.moves.length > 0) {
          await apply(
            to,
            () => UpdateEntries(gameId, to.id, plan.moves),
            plural(
              { count: plan.moves.length },
              {
                one: `Matched # version in ${{ name: to.name }}`,
                other: `Matched # versions in ${{ name: to.name }}`,
              },
            ),
          )
        }
        await refresh()
      },
      { errorTitle: t`Could not change ${{ name: to.name }}` },
    )
  }
  return { pending, lockedReason, matchTo }
}

function CompareFooter({
  profileA,
  profileB,
  selected,
  pending,
  lockedReason,
  matchTo,
}: {
  profileA: Profile
  profileB: Profile
  selected: CompareRow[]
  pending: boolean
  lockedReason: (profile: Profile) => string
  matchTo: (toB: boolean) => void
}) {
  const { t } = useLingui()
  const count = new Set(selected.map((row) => row.id)).size
  const actions = [
    {
      toB: true,
      icon: <ArrowRight size={16} />,
      label: t`Make ${{ name: profileB.name }} match ${{ other: profileA.name }}`,
    },
    {
      toB: false,
      icon: <ArrowLeft size={16} />,
      label: t`Make ${{ name: profileA.name }} match ${{ other: profileB.name }}`,
    },
  ]
  return (
    <DialogFooter aside={t`${count} selected`}>
      {actions.map(({ toB, icon, label }) => {
        const target = toB ? profileB : profileA
        const plan = applyPlan(selected, toB)
        const idle = plan.copy.length + plan.moves.length === 0
        const reason =
          lockedReason(target) ||
          (idle ? t`Nothing selected would change ${{ name: target.name }}.` : '')
        return (
          <DisabledReason key={String(toB)} title={reason} disabled={reason !== ''}>
            <Button
              variant={toB ? 'text' : 'contained'}
              startIcon={icon}
              disabled={pending || reason !== ''}
              onClick={() => matchTo(toB)}
              sx={{ whiteSpace: 'nowrap' }}
            >
              {label}
            </Button>
          </DisabledReason>
        )
      })}
    </DialogFooter>
  )
}

function Summary({
  view,
  aName,
  shown,
  onToggle,
}: {
  view: CompareView
  aName: string
  shown: ReadonlySet<HideableKind>
  onToggle: (kind: HideableKind) => void
}) {
  const { t } = useLingui()
  const hideable: { kind: HideableKind; label: string }[] = [
    { kind: 'onlyA', label: t`${{ onlyA: view.onlyA }} only in ${{ name: aName }}` },
    { kind: 'identical', label: t`${{ identical: view.identical }} identical` },
  ]
  return (
    <Typography
      component="div"
      sx={{
        px: 3,
        pb: 2,
        display: 'flex',
        alignItems: 'baseline',
        flexWrap: 'wrap',
        gap: space.gap,
      }}
    >
      <span>
        {view.differences === 0
          ? t`No differences.`
          : plural({ count: view.differences }, { one: '# difference', other: '# differences' })}
      </span>
      {hideable
        .filter(({ kind }) => view[kind] > 0)
        .map(({ kind, label }) => (
          <Typography key={kind} component="span" color="text.secondary">
            {label}{' '}
            <Button size="small" onClick={() => onToggle(kind)} sx={{ minWidth: 0, p: 0.25 }}>
              {shown.has(kind) ? t`Hide` : t`Show`}
            </Button>
          </Typography>
        ))}
    </Typography>
  )
}

const COMPARE_HEIGHT = 600

function CompareFrame({
  open,
  onClose,
  footer,
  children,
}: {
  open: boolean
  onClose: () => void
  footer?: ReactNode
  children: ReactNode
}) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.name ?? '')
  return (
    <AppDialog
      closeButton={true}
      open={open}
      onClose={onClose}
      title={t`Compare`}
      subtitle={game}
      icon={
        <ToneIcon tone="neutral">
          <GitCompare size={22} />
        </ToneIcon>
      }
      size="xl"
      height={COMPARE_HEIGHT}
      flush={true}
      footer={footer}
    >
      {children}
    </AppDialog>
  )
}

interface Picks {
  onPick: (id: string) => void
  onFriend: () => void
  onHost: () => void
}

function ComparePair({
  profileA,
  profileB,
  open,
  onClose,
  onPick,
  onFriend,
  onHost,
}: Picks & { profileA: Profile; profileB: Profile; open: boolean; onClose: () => void }) {
  const { t } = useLingui()
  const [shown, setShown] = useState<ReadonlySet<HideableKind>>(new Set())
  const [filterOpen, setFilterOpen] = useState(false)
  const [filter, setFilter] = useState('')
  const [toggled, setToggled] = useState<ReadonlySet<string>>(new Set())

  const diff = useMemo(() => compareProfiles(profileA, profileB), [profileA, profileB])
  const needle = filter.trim().toLowerCase()
  const view = compareView(diff, needle, shown)
  const groupTitle = useGroupTitle(profileA.name, profileB.name)

  const keyOf = (row: CompareRow) => `${row.kind}:${row.id}`
  const isChecked = (row: CompareRow) => checkedByDefault(row) !== toggled.has(keyOf(row))
  const toggle = (row: CompareRow) =>
    setToggled((prev) => {
      const next = new Set(prev)
      if (!next.delete(keyOf(row))) {
        next.add(keyOf(row))
      }
      return next
    })
  const selected = view.groups
    .flatMap((g) => g.rows)
    .filter((row) => row.kind !== 'identical' && isChecked(row))
  const { pending, lockedReason, matchTo } = useMatch(profileA, profileB, selected)

  return (
    <CompareFrame
      open={open}
      onClose={pending ? () => undefined : onClose}
      footer={
        <CompareFooter
          profileA={profileA}
          profileB={profileB}
          selected={selected}
          pending={pending}
          lockedReason={lockedReason}
          matchTo={matchTo}
        />
      }
    >
      <CompareTop
        profileA={profileA}
        profileB={profileB}
        pending={pending}
        filterOpen={filterOpen}
        onFilter={() => setFilterOpen((v) => !v)}
        onPick={(id) => {
          setToggled(new Set())
          onPick(id)
        }}
        onFriend={onFriend}
        onHost={onHost}
      />
      {filterOpen ? (
        <Box sx={{ px: 3, pb: 2 }}>
          <SearchField
            label={t`Filter mods`}
            fullWidth={true}
            autoFocus={true}
            value={filter}
            onChange={setFilter}
          />
        </Box>
      ) : null}
      <Summary
        view={view}
        aName={profileA.name}
        shown={shown}
        onToggle={(kind) =>
          setShown((prev) => {
            const next = new Set(prev)
            if (!next.delete(kind)) {
              next.add(kind)
            }
            return next
          })
        }
      />
      <Box
        sx={{
          flex: 1,
          minHeight: 0,
          overflowY: 'auto',
          px: 3,
          pt: space.gap,
          borderTop: 1,
          borderColor: 'divider',
        }}
      >
        {view.groups.length === 0 ? (
          <EmptyState
            compact={true}
            icon={<ListChecks size={28} />}
            title={needle ? t`No matching mods` : t`These profiles have the same mods.`}
          >
            {needle ? t`Neither profile has a mod that matches the filter.` : ''}
          </EmptyState>
        ) : (
          <>
            <CompareHeader aName={profileA.name} bName={profileB.name} />
            {view.groups.map((group) => (
              <CompareGroupView
                key={group.kind}
                group={group}
                title={groupTitle(group.kind)}
                profileA={profileA}
                profileB={profileB}
                isChecked={isChecked}
                onToggle={toggle}
                disabled={pending}
              />
            ))}
          </>
        )}
      </Box>
    </CompareFrame>
  )
}

function CompareChoose({
  profileA,
  open,
  onClose,
  onPick,
  onFriend,
  onHost,
}: Picks & { profileA: Profile; open: boolean; onClose: () => void }) {
  const { t } = useLingui()
  const listed = useProfiles((s) => hasModList(s.game ?? undefined))
  return (
    <CompareFrame open={open} onClose={onClose}>
      <CompareTop
        profileA={profileA}
        profileB={null}
        pending={false}
        filterOpen={false}
        onFilter={() => undefined}
        onPick={onPick}
        onFriend={onFriend}
        onHost={onHost}
      />
      <Box sx={{ flex: 1, display: 'flex', borderTop: 1, borderColor: 'divider' }}>
        <EmptyState
          compact={true}
          icon={<GitCompare size={28} />}
          title={t`Pick what to compare ${{ name: profileA.name }} with`}
          action={
            <Box
              sx={{ display: 'flex', gap: space.gap, flexWrap: 'wrap', justifyContent: 'center' }}
            >
              <Button
                variant="outlined"
                color="inherit"
                startIcon={<Users size={16} />}
                onClick={onFriend}
              >
                {t`A friend's link or file…`}
              </Button>
              {listed ? (
                <Button
                  variant="outlined"
                  color="inherit"
                  startIcon={<Users size={16} />}
                  onClick={onHost}
                >
                  {t`A multiplayer host's list…`}
                </Button>
              ) : null}
            </Box>
          }
        >
          {t`Compare against a friend's profile or a multiplayer host's list.`}
        </EmptyState>
      </Box>
    </CompareFrame>
  )
}

function CompareBody({ a, b, onClose }: { a: Profile; b: Profile | null; onClose: () => void }) {
  const listed = useProfiles((s) => s.profiles)
  const [otherId, setOtherId] = useState(b?.id ?? null)
  const [farm, setFarm] = useState(false)

  const profileA = listed.find((p) => p.id === a.id) ?? a
  const profileB = listed.find((p) => p.id === otherId) ?? b
  const shared = {
    profileA,
    open: !farm,
    onClose,
    onPick: setOtherId,
    onFriend: () => {
      onClose()
      openImport({ profileId: profileA.id })
    },
    onHost: () => setFarm(true),
  }
  return (
    <>
      {profileB ? <ComparePair {...shared} profileB={profileB} /> : <CompareChoose {...shared} />}
      <FarmDialog open={farm} profile={profileA} onClose={() => setFarm(false)} />
    </>
  )
}

export function CompareDialog({
  a,
  b,
  onClose,
}: {
  a: Profile | null
  b?: Profile | null
  onClose: () => void
}) {
  if (!a) {
    return null
  }
  return <CompareBody a={a} b={b ?? null} onClose={onClose} />
}
