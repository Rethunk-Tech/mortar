import { Box, Switch, Typography } from '@mui/material'
import { ChevronDown, ChevronRight } from 'lucide-react'
import type { ReactNode } from 'react'
import { CoverButton } from '../shell/CoverButton.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { GroupMenu } from './GroupMenu.tsx'
import { toggleCollapsed } from './group.ts'
import { LockedReason } from './LockedReason.tsx'
import type { ListRow } from './listColumns.ts'
import { useMods } from './store.ts'
import { setGroupEnabled } from './storeEntries.ts'
import { useLocked } from './useLocked.ts'

function ModsGroupHeader({
  label,
  count,
  open,
  onToggle,
  hint,
  enabled,
  onEnabled,
  menu,
}: {
  label: string
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
        px: 2,
        py: 0.75,
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
      <Typography sx={{ fontSize: 13, fontWeight: 600 }}>{label}</Typography>
      <Box component="span" sx={{ fontSize: 12, color: 'text.secondary' }}>
        {count}
      </Box>
      {onEnabled ? (
        <Box
          data-control={true}
          sx={{ ml: 'auto', position: 'relative', display: 'flex', alignItems: 'center' }}
        >
          {menu}
          <LockedReason locked={locked}>
            <Switch
              size="small"
              checked={enabled === true}
              disabled={locked}
              onChange={(_, on) => onEnabled(on)}
              slotProps={{ input: { 'aria-label': label } }}
            />
          </LockedReason>
        </Box>
      ) : null}
    </Box>
  )
}

function GroupHeaderRow({
  groupKey,
  count,
  label,
  collapsed,
  gameId,
  setCollapsed,
  groupBy,
  tagHint,
  groups,
}: {
  groupKey: string
  count: number
  label: string
  collapsed: Record<string, boolean>
  gameId: string
  setCollapsed: (fn: (cur: Record<string, boolean>) => Record<string, boolean>) => void
  groupBy: string
  tagHint: string
  groups: readonly { key: string; items: readonly ListRow[] }[]
}) {
  const open = collapsed[groupKey] !== true
  return (
    <ModsGroupHeader
      label={label}
      count={count}
      open={open}
      onToggle={() => setCollapsed((cur) => toggleCollapsed(gameId, cur, groupKey, open))}
      {...(groupBy === 'tag' ? { hint: tagHint } : {})}
      {...(groupBy === 'group' && groupKey !== ''
        ? {
            enabled:
              groups.find((g) => g.key === groupKey)?.items.every((r) => r.mod.enabled) === true,
            menu: <GroupMenu name={groupKey} />,
            onEnabled: (on: boolean) => {
              setGroupEnabled(groupKey, on)
                .then(() => useMods.getState().load())
                .catch(reportUnexpected)
            },
          }
        : {})}
    />
  )
}

export { GroupHeaderRow }
