import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  IconButton,
  ListItemIcon,
  ListItemText,
  ListSubheader,
  Menu,
  MenuItem,
  Typography,
} from '@mui/material'
import {
  ArrowLeft,
  ArrowRight,
  Check,
  ChevronDown,
  ListChecks,
  Search,
  Users,
  X,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  CopyMods,
  UpdateEntries,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { FarmDialog } from '../game/FarmMenuItem.tsx'
import { applyWithUndo } from '../mods/menu.ts'
import { lockedIn, useLaunchLocks } from '../mods/useLocked.ts'
import { openImport } from '../share/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { useToasts } from '../toasts/store.ts'
import { pushUndoToast } from '../toasts/undo.ts'
import { usePending } from '../toasts/usePending.ts'
import { colorHex } from './appearance.ts'
import { CompareGroupView, CompareHeader } from './CompareTable.tsx'
import type { CompareRow, CompareView } from './compare.ts'
import { applyPlan, compareProfiles, compareView } from './compare.ts'
import { useGroupTitle } from './compareTitle.ts'
import { userModCount } from './count.ts'
import { useProfiles } from './store.ts'

function Dot({ profile }: { profile: Profile }) {
  return (
    <Box
      component="span"
      aria-hidden={true}
      sx={{
        width: 10,
        height: 10,
        borderRadius: '50%',
        flexShrink: 0,
        bgcolor: colorHex(profile.color) ?? 'primary.main',
      }}
    />
  )
}

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
      ? t`${profile.name} is in use by the running game. Close the game to change its mods.`
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
            plural(plan.copy.length, {
              one: `Copied # mod to ${to.name}`,
              other: `Copied # mods to ${to.name}`,
            }),
          )
        }
        if (plan.moves.length > 0) {
          await apply(
            to,
            () => UpdateEntries(gameId, to.id, plan.moves),
            plural(plan.moves.length, {
              one: `Matched # version in ${to.name}`,
              other: `Matched # versions in ${to.name}`,
            }),
          )
        }
        await refresh()
      },
      { errorTitle: t`Could not change ${to.name}` },
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
      label: t`Make ${profileB.name} match ${profileA.name}`,
    },
    {
      toB: false,
      icon: <ArrowLeft size={16} />,
      label: t`Make ${profileA.name} match ${profileB.name}`,
    },
  ]
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1.25,
        px: 3,
        py: 1.75,
        borderTop: 1,
        borderColor: 'divider',
      }}
    >
      <Typography color="text.secondary" sx={{ fontSize: 13 }}>
        {t`${count} selected`}
      </Typography>
      <Box sx={{ flex: 1 }} />
      {actions.map(({ toB, icon, label }) => {
        const reason = lockedReason(toB ? profileB : profileA)
        const plan = applyPlan(selected, toB)
        return (
          <DisabledReason key={String(toB)} title={reason} disabled={reason !== ''}>
            <Button
              variant={toB ? 'outlined' : 'contained'}
              color={toB ? 'inherit' : 'primary'}
              startIcon={icon}
              disabled={pending || reason !== '' || plan.copy.length + plan.moves.length === 0}
              onClick={() => matchTo(toB)}
              sx={{ whiteSpace: 'nowrap' }}
            >
              {label}
            </Button>
          </DisabledReason>
        )
      })}
    </Box>
  )
}

function OtherPicker({
  profileA,
  profileB,
  gameId,
  disabled,
  onPick,
  onFriend,
  onHost,
}: {
  profileA: Profile
  profileB: Profile
  gameId: string
  disabled: boolean
  onPick: (id: string) => void
  onFriend: () => void
  onHost: () => void
}) {
  const { t } = useLingui()
  const listed = useProfiles((s) => s.profiles)
  const [menu, setMenu] = useState<HTMLElement | null>(null)
  const heading = { fontSize: 11, fontWeight: 700, letterSpacing: '0.08em', lineHeight: '28px' }
  const choose = (action: () => void) => () => {
    setMenu(null)
    action()
  }
  return (
    <>
      <Button
        color="inherit"
        variant="outlined"
        disabled={disabled}
        aria-haspopup="menu"
        onClick={(e) => setMenu(e.currentTarget)}
        endIcon={<ChevronDown size={14} />}
        sx={{ height: 36, whiteSpace: 'nowrap' }}
      >
        {profileB.name}
      </Button>
      <Menu anchorEl={menu} open={menu !== null} onClose={() => setMenu(null)}>
        <ListSubheader sx={heading}>{t`YOUR PROFILES`}</ListSubheader>
        {listed
          .filter((p) => p.id !== profileA.id)
          .map((p) => (
            <MenuItem key={p.id} onClick={choose(() => onPick(p.id))}>
              <ListItemIcon>
                <Dot profile={p} />
              </ListItemIcon>
              <ListItemText>{t`${p.name} · ${userModCount(p)} mods`}</ListItemText>
              {p.id === profileB.id ? <Check size={16} aria-hidden={true} /> : null}
            </MenuItem>
          ))}
        <ListSubheader sx={heading}>{t`SOMEONE ELSE'S`}</ListSubheader>
        <MenuItem onClick={choose(onFriend)}>
          <ListItemIcon>
            <Users size={16} />
          </ListItemIcon>
          <ListItemText>{t`A friend's link or file…`}</ListItemText>
        </MenuItem>
        {gameId === 'stardew' ? (
          <MenuItem onClick={choose(onHost)}>
            <ListItemIcon>
              <Users size={16} />
            </ListItemIcon>
            <ListItemText>{t`A multiplayer host's list…`}</ListItemText>
          </MenuItem>
        ) : null}
      </Menu>
    </>
  )
}

function Summary({
  view,
  aName,
  showAll,
  onToggle,
}: {
  view: CompareView
  aName: string
  showAll: boolean
  onToggle: () => void
}) {
  const { t } = useLingui()
  const { onlyA, identical } = view
  const rest = showAll
    ? t` · ${onlyA} only in ${aName}, ${identical} identical, shown`
    : t` · ${onlyA} only in ${aName}, ${identical} identical, hidden`
  return (
    <Typography component="div" sx={{ px: 3, pb: 1.5 }}>
      {view.differences === 0
        ? t`No differences.`
        : plural(view.differences, { one: '# difference', other: '# differences' })}
      {onlyA + identical > 0 ? (
        <>
          <Typography component="span" color="text.secondary">
            {rest}
          </Typography>{' '}
          <Button size="small" onClick={onToggle} sx={{ minWidth: 0, p: 0.25 }}>
            {showAll ? t`Hide` : t`Show`}
          </Button>
        </>
      ) : null}
    </Typography>
  )
}

function CompareTop({
  profileA,
  profileB,
  pending,
  filterOpen,
  onFilter,
  onClose,
  onPick,
  onHost,
}: {
  profileA: Profile
  profileB: Profile
  pending: boolean
  filterOpen: boolean
  onFilter: () => void
  onClose: () => void
  onPick: (id: string) => void
  onHost: () => void
}) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game)
  const gameId = game?.id ?? ''
  return (
    <>
      <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 1.25, px: 3, pt: 2.5, pb: 2 }}>
        <Typography component="h2" sx={{ fontSize: 20, fontWeight: 700 }}>
          {t`Compare`}
        </Typography>
        <Typography color="text.secondary">{game?.name ?? ''}</Typography>
        <Box sx={{ flex: 1 }} />
        <IconButton size="small" aria-label={t`Close`} onClick={onClose} disabled={pending}>
          <X size={18} />
        </IconButton>
      </Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, px: 3, pb: 2 }}>
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: 1,
            height: 36,
            px: 1.5,
            borderRadius: '8px',
            bgcolor: 'var(--mortar-hairline-12)',
          }}
        >
          <Dot profile={profileA} />
          <b>{profileA.name}</b>
        </Box>
        <Typography color="text.secondary">{t`with`}</Typography>
        <OtherPicker
          profileA={profileA}
          profileB={profileB}
          gameId={gameId}
          disabled={pending}
          onPick={onPick}
          onFriend={() => {
            onClose()
            openImport({ profileId: profileA.id })
          }}
          onHost={onHost}
        />
        <Box sx={{ flex: 1 }} />
        <IconButton
          aria-label={t`Filter mods`}
          aria-pressed={filterOpen}
          onClick={() => onFilter()}
          sx={{ width: 36, height: 36, borderRadius: '8px', border: 1, borderColor: 'divider' }}
        >
          <Search size={16} />
        </IconButton>
      </Box>
    </>
  )
}

function CompareBody({ a, b, onClose }: { a: Profile; b: Profile; onClose: () => void }) {
  const { t } = useLingui()
  const listed = useProfiles((s) => s.profiles)
  const [otherId, setOtherId] = useState(b.id)
  const [farm, setFarm] = useState(false)
  const [showAll, setShowAll] = useState(false)
  const [filterOpen, setFilterOpen] = useState(false)
  const [filter, setFilter] = useState('')
  const [toggled, setToggled] = useState<ReadonlySet<string>>(new Set())

  const profileA = listed.find((p) => p.id === a.id) ?? a
  const profileB = listed.find((p) => p.id === otherId) ?? b
  const diff = useMemo(() => compareProfiles(profileA, profileB), [profileA, profileB])
  const needle = filter.trim().toLowerCase()
  const view = compareView(diff, needle, showAll)
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
    <>
      <Dialog
        open={!farm}
        onClose={pending ? undefined : onClose}
        slotProps={{
          paper: {
            sx: {
              width: 880,
              maxWidth: 'calc(100vw - 64px)',
              height: 600,
              maxHeight: 'calc(100vh - 64px)',
            },
          },
        }}
      >
        <CompareTop
          profileA={profileA}
          profileB={profileB}
          pending={pending}
          filterOpen={filterOpen}
          onFilter={() => setFilterOpen((v) => !v)}
          onClose={onClose}
          onPick={(id) => {
            setOtherId(id)
            setToggled(new Set())
          }}
          onHost={() => setFarm(true)}
        />
        {filterOpen ? (
          <Box sx={{ px: 3, pb: 1.5 }}>
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
          showAll={showAll}
          onToggle={() => setShowAll((v) => !v)}
        />
        <Box
          sx={{
            flex: 1,
            minHeight: 0,
            overflowY: 'auto',
            px: 3,
            pt: 1,
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
        <CompareFooter
          profileA={profileA}
          profileB={profileB}
          selected={selected}
          pending={pending}
          lockedReason={lockedReason}
          matchTo={matchTo}
        />
      </Dialog>
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
  b: Profile | null
  onClose: () => void
}) {
  if (!(a && b)) {
    return null
  }
  return <CompareBody a={a} b={b} onClose={onClose} />
}
