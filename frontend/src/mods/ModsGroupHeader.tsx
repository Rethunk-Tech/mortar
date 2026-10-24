import { useLingui } from '@lingui/react/macro'
import { Box, Button, Switch, Typography } from '@mui/material'
import { ChevronDown, ChevronRight, Search } from 'lucide-react'
import { memo, type ReactNode } from 'react'
import { useProfiles } from '../profiles/store.ts'
import { CoverButton } from '../shell/CoverButton.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { GroupMenu } from './GroupMenu.tsx'
import { groupDepth, inGroup, toggleCollapsed } from './group.ts'
import { LockedReason } from './LockedReason.tsx'
import type { ListRow } from './listColumns.ts'
import { useMods } from './store.ts'
import { findSources } from './storeEntries.ts'
import { setGroupEnabled } from './storeGroups.ts'
import { useLocked } from './useLocked.ts'
import { GROUP_HEADER_PX } from './virtualRows.ts'

// How far each level of a nested group is set in.
const NEST_INDENT_PX = 16

function ModsGroupHeader({
  label,
  count,
  open,
  onToggle,
  hint,
  enabled,
  onEnabled,
  menu,
  depth = 0,
}: {
  label: string
  depth?: number
  count: number
  open: boolean
  onToggle: () => void
  hint?: string
  enabled?: boolean
  onEnabled?: (on: boolean) => void
  menu?: ReactNode
}) {
  const locked = useLocked()
  return (
    <Box
      sx={{
        position: 'relative',
        display: 'flex',
        alignItems: 'center',
        gap: 0.75,
        width: '100%',
        // The virtual list reserves one height per header, and a header's switch and menu fit inside it.
        minHeight: GROUP_HEADER_PX,
        px: space.pad,
        pl: depth > 0 ? `calc(${space.pad}px + ${depth * NEST_INDENT_PX}px)` : space.pad,
        bgcolor: 'var(--mortar-hairline-ghost)',
        borderBottom: '1px solid var(--mortar-hairline-muted)',
        '& > :not(.MuiButtonBase-root, [data-control])': { pointerEvents: 'none' },
      }}
    >
      <CoverButton onClick={onToggle} aria-expanded={open} aria-label={label} title={hint} />
      {open ? (
        <ChevronDown size={14} aria-hidden={true} />
      ) : (
        <ChevronRight size={14} aria-hidden={true} />
      )}
      <Typography sx={{ fontSize: 'base.fontSize', fontWeight: 600 }}>{label}</Typography>
      <Box component="span" sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
        {count}
      </Box>
      {menu || onEnabled ? (
        <Box
          data-control={true}
          sx={{ ml: 'auto', position: 'relative', display: 'flex', alignItems: 'center' }}
        >
          {menu}
          {onEnabled ? (
            <LockedReason locked={locked}>
              <Switch
                size="small"
                checked={enabled === true}
                disabled={locked}
                onChange={(_, on) => onEnabled(on)}
                slotProps={{ input: { 'aria-label': label } }}
              />
            </LockedReason>
          ) : null}
        </Box>
      ) : null}
    </Box>
  )
}

// The button on a source group that holds mods with no recorded site: looks each one up on the sites the game has a
// matcher for.
function FindSourcesButton() {
  const { t } = useLingui()
  const locked = useLocked()
  const [pending, run] = usePending()
  return (
    <LockedReason locked={locked}>
      <Button
        size="small"
        // The list gives every group header one fixed height, which a small button's own padding overflows.
        sx={{ height: 20, minHeight: 0, py: 0 }}
        disabled={locked || pending}
        startIcon={<Search size={14} aria-hidden={true} />}
        onClick={() => run(findSources, { errorTitle: t`Could not look these mods up` })}
      >
        {t`Find sources`}
      </Button>
    </LockedReason>
  )
}

interface HeaderRowProps {
  groupKey: string
  count: number
  label: string
  collapsed: Record<string, boolean>
  gameId: string
  setCollapsed: (fn: (cur: Record<string, boolean>) => Record<string, boolean>) => void
  groupBy: string
  tagHint: string
  groups: readonly { key: string; items: readonly ListRow[] }[]
}

// Whether every mod of the group and of the groups inside it is on; a group with no mods at all reads as off.
function subtreeEnabled(groups: HeaderRowProps['groups'], key: string) {
  const rows = groups.filter((g) => inGroup(g.key, key)).flatMap((g) => g.items)
  return rows.length > 0 && rows.every((r) => r.mod.enabled)
}

function GroupHeaderRowView({
  groupKey,
  count,
  label,
  collapsed,
  gameId,
  setCollapsed,
  groupBy,
  tagHint,
  groups,
}: HeaderRowProps) {
  const open = collapsed[groupKey] !== true
  return (
    <ModsGroupHeader
      label={label}
      depth={groupBy === 'group' ? groupDepth(groupKey) : 0}
      count={count}
      open={open}
      onToggle={() => setCollapsed((cur) => toggleCollapsed(gameId, cur, groupKey, open))}
      {...(groupBy === 'tag' ? { hint: tagHint } : {})}
      {...(groupBy === 'group' && groupKey !== ''
        ? {
            enabled: subtreeEnabled(groups, groupKey),
            menu: <GroupMenu name={groupKey} />,
            onEnabled: (on: boolean) => {
              setGroupEnabled(groupKey, on)
                .then(() => useMods.getState().load())
                .catch(reportUnexpected)
            },
          }
        : {})}
      {...(groupBy === 'source' &&
      (useProfiles.getState().game?.findSources?.length ?? 0) > 0 &&
      groups.find((g) => g.key === groupKey)?.items.some((r) => r.unsourced === true) === true
        ? { menu: <FindSourcesButton /> }
        : {})}
    />
  )
}

// Of the collapsed map and the groups, a header shows only whether its group is collapsed, all switched on, or holds an
// unsourced mod; the list rebuilds both for any change to any row, and a header redraws only when these change.
const headerFlags = (p: HeaderRowProps) => {
  const group = p.groups.find((g) => g.key === p.groupKey)
  return [
    p.collapsed[p.groupKey] === true,
    subtreeEnabled(p.groups, p.groupKey),
    group?.items.some((r) => r.unsourced === true) === true,
  ].join()
}

const GroupHeaderRow = memo(
  GroupHeaderRowView,
  (a, b) =>
    a.groupKey === b.groupKey &&
    a.count === b.count &&
    a.label === b.label &&
    a.gameId === b.gameId &&
    a.setCollapsed === b.setCollapsed &&
    a.groupBy === b.groupBy &&
    a.tagHint === b.tagHint &&
    headerFlags(a) === headerFlags(b),
)

// The open group's header pinned over the top of a scrolled list, on a solid surface so rows do not show through it.
function StickyGroupSlot({ shift, children }: { shift: number; children: ReactNode }) {
  return (
    <Box sx={{ position: 'sticky', top: 0, height: 0, zIndex: 2 }}>
      <Box sx={{ transform: `translateY(${shift}px)`, bgcolor: 'var(--mortar-menu)' }}>
        {children}
      </Box>
    </Box>
  )
}

export { GroupHeaderRow, StickyGroupSlot }
