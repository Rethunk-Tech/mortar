import { useLingui } from '@lingui/react/macro'
import {
  alpha,
  Box,
  Table,
  TableBody,
  TableCell,
  type TableCellProps,
  TableHead,
  TableRow,
  useMediaQuery,
} from '@mui/material'
import { ArrowDown, ArrowUp, Pin } from 'lucide-react'
import { type MouseEvent, type ReactNode, useEffect, useState } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  SetListColumns,
  SetListSort,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { compactQuery } from '../game/compact.ts'
import { useSettings } from '../settings/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useDetail } from './detail.ts'
import {
  emptyGroupLabel,
  firstTag,
  type GroupBy,
  groupSorted,
  sanitizeListGroupBy,
} from './group.ts'
import { ListColumnMenu } from './ListColumnMenu.tsx'
import {
  columnMenuFromEvent,
  compareListRows,
  DEFAULT_VISIBLE_LIST_COLUMNS,
  type ListColumnId,
  type ListRow,
  listGridColumns,
  nextListSort,
  sanitizeListColumns,
  sanitizeListSort,
  toggleListColumn,
  visibleListColumns,
} from './listColumns.ts'
import { toListRow } from './listRows.ts'
import { modId, nexusIdOf } from './lookup.ts'
import { ModsGroupHeader } from './ModsGroupHeader.tsx'
import { contextMenuProps } from './menu.ts'
import { primeDetails, useNexusDetails, useNexusFresh } from './nexusDetails.ts'
import { formatCount, formatDate, isNewer } from './nexusFormat.ts'
import { heading } from './paper.ts'
import { LetterTile, ModSwitch, PinBadge, ProblemBadge, UpdateBadge } from './parts.tsx'
import { useSelection } from './selection.ts'

const SELECTED_ALPHA = 0.14
const ellipsis = { overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } as const

const cellBase = { p: 0, border: 0, fontSize: 'inherit', color: 'inherit' } as const

function Cell({ sx, ...props }: TableCellProps) {
  return <TableCell {...props} sx={{ ...cellBase, ...sx }} />
}

function headerLabel(id: ListColumnId, t: ReturnType<typeof useLingui>['t']): string {
  switch (id) {
    case 'on':
      return t`On`
    case 'name':
      return t`Name`
    case 'version':
      return t`Version`
    case 'latest':
      return t`Latest on Nexus`
    case 'uniqueId':
      return t`UniqueID`
    case 'author':
      return t`Author`
    case 'source':
      return t`Source`
    case 'category':
      return t`Category`
    case 'endorsements':
      return t`Endorsements`
    case 'downloads':
      return t`Downloads`
    case 'updated':
      return t`Updated on Nexus`
    case 'installed':
      return t`Installed`
    case 'needs':
      return t`Needs`
    case 'status':
      return t`Status`
    case 'notes':
      return t`Notes and tags`
    default:
      return id
  }
}

function dash(value: string) {
  return value || '—'
}

function HeaderCell({
  id,
  label,
  sortColumn,
  sortDir,
  onSort,
  onMenu,
}: {
  id: ListColumnId
  label: string
  sortColumn: ListColumnId
  sortDir: 'asc' | 'desc'
  onSort: (id: ListColumnId) => void
  onMenu: (e: MouseEvent) => void
}) {
  const active = id !== 'on' && sortColumn === id
  return (
    <Cell
      title={label}
      onClick={id === 'on' ? undefined : () => onSort(id)}
      onContextMenu={onMenu}
      sx={{
        ...ellipsis,
        cursor: id === 'on' ? 'default' : 'pointer',
        userSelect: 'none',
        display: 'flex',
        alignItems: 'center',
        gap: 0.5,
      }}
    >
      {label}
      {active && sortDir === 'asc' ? <ArrowUp size={12} aria-hidden={true} /> : null}
      {active && sortDir === 'desc' ? <ArrowDown size={12} aria-hidden={true} /> : null}
    </Cell>
  )
}

function ValueCell({ text, accent }: { text: string; accent?: boolean }) {
  return (
    <Cell
      title={text}
      sx={{
        ...ellipsis,
        color: accent ? 'primary.main' : 'text.secondary',
        fontVariantNumeric: 'tabular-nums',
      }}
    >
      {text}
    </Cell>
  )
}

function cellsFor(id: ListColumnId, row: ListRow, locale: string) {
  const m = row.mod
  const page = row.details?.page
  switch (id) {
    case 'on':
      return (
        <Cell key="on">
          <ModSwitch mod={m} />
        </Cell>
      )
    case 'name':
      return (
        <Cell key="name" title={m.name} sx={{ ...ellipsis, fontWeight: 500 }}>
          {m.name}
        </Cell>
      )
    case 'version':
      return (
        <Cell key="version" title={dash(m.version)} sx={{ ...ellipsis, color: 'text.secondary' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, minWidth: 0 }}>
            <Box component="span" sx={{ ...ellipsis, fontVariantNumeric: 'tabular-nums' }}>
              {dash(m.version)}
            </Box>
            {row.pinned ? <Pin size={14} aria-hidden={true} /> : null}
          </Box>
        </Cell>
      )
    case 'latest': {
      const latest = page?.version ?? ''
      return (
        <ValueCell
          key="latest"
          text={dash(latest)}
          accent={latest !== '' && isNewer(latest, m.version)}
        />
      )
    }
    case 'uniqueId':
      return <ValueCell key="uniqueId" text={dash(m.uniqueId)} />
    case 'author':
      return <ValueCell key="author" text={dash(m.author)} />
    case 'source':
      return <ValueCell key="source" text={dash(row.source)} />
    case 'category':
      return <ValueCell key="category" text={dash(row.details?.category ?? '')} />
    case 'endorsements': {
      const n = page?.endorsements ?? m.endorsements
      const missing = page === undefined && m.endorsements === 0
      return <ValueCell key="endorsements" text={missing ? '—' : formatCount(n, locale)} />
    }
    case 'downloads':
      return (
        <ValueCell
          key="downloads"
          text={page === undefined ? '—' : formatCount(page.downloads, locale)}
        />
      )
    case 'updated':
      return <ValueCell key="updated" text={dash(formatDate(page?.updated ?? '', locale))} />
    case 'installed':
      return <ValueCell key="installed" text={dash(formatDate(row.added, locale))} />
    case 'needs': {
      const names = m.needs
      if (names === undefined || names === null) {
        return <ValueCell key="needs" text="—" />
      }
      const text = names.length === 0 ? '0' : names.join(', ')
      return <ValueCell key="needs" text={text} />
    }
    case 'status':
      return (
        <Cell
          key="status"
          sx={{
            fontSize: 13,
            whiteSpace: 'nowrap',
            color: m.enabled ? 'text.primary' : 'text.secondary',
          }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, minWidth: 0 }}>
            <PinBadge mod={m} />
            <ProblemBadge mod={m} />
            <UpdateBadge mod={m} />
            <Box component="span" title={row.status} sx={ellipsis}>
              {row.status}
            </Box>
          </Box>
        </Cell>
      )
    case 'notes': {
      const text = [row.note, row.tags.join(', ')].filter((part) => part !== '').join(' · ')
      return <ValueCell key="notes" text={dash(text)} />
    }
    default:
      return null
  }
}

function ModRow({
  row,
  striped,
  cols,
  locale,
  orderedIds,
  profile,
}: {
  row: ListRow
  striped: boolean
  cols: readonly ListColumnId[]
  locale: string
  orderedIds: readonly string[]
  profile: Profile
}) {
  const detailId = useDetail((s) => s.detailId)
  const selectedIds = useSelection((s) => s.ids)
  const show = useDetail((s) => s.show)
  const m = row.mod
  const rowId = modId(m)
  const marked = selectedIds.includes(rowId) || (selectedIds.length === 0 && rowId === detailId)
  const menu = contextMenuProps(m)
  const fresh = useNexusFresh(nexusIdOf(profile, m))
  return (
    <TableRow
      hover={true}
      selected={marked}
      onMouseDown={(e) => {
        if (e.shiftKey) {
          e.preventDefault()
        }
      }}
      onClick={(e) => {
        useSelection.getState().click(orderedIds, rowId, e)
        show(m)
      }}
      tabIndex={0}
      {...menu}
      onKeyDown={menu.onKeyDown}
      sx={{
        display: 'grid',
        gridTemplateColumns: listGridColumns(cols),
        gap: '10px',
        alignItems: 'center',
        px: 2,
        height: 36,
        fontSize: 14,
        cursor: 'pointer',
        bgcolor: (th) => {
          if (marked) {
            return alpha(th.palette.primary.main, SELECTED_ALPHA)
          }
          return striped ? 'rgba(255,255,255,0.03)' : 'transparent'
        },
      }}
    >
      {cols.flatMap((id) =>
        id === 'name'
          ? [
              <Cell key="tile">
                <LetterTile mod={m} size={26} fresh={fresh} />
              </Cell>,
              cellsFor(id, row, locale),
            ]
          : [cellsFor(id, row, locale)],
      )}
    </TableRow>
  )
}

function persistColumns(ids: ListColumnId[]) {
  useSettings.setState({ listColumns: ids })
  SetListColumns(ids).catch(reportUnexpected)
}

function persistSort(column: ListColumnId, dir: 'asc' | 'desc') {
  useSettings.setState({ listSortColumn: column, listSortDir: dir })
  SetListSort(column, dir).catch(reportUnexpected)
}

function HeaderCells({
  cols,
  sort,
  t,
  onMenu,
}: {
  cols: readonly ListColumnId[]
  sort: ReturnType<typeof sanitizeListSort>
  t: ReturnType<typeof useLingui>['t']
  onMenu: (e: MouseEvent) => void
}) {
  const onSort = (col: ListColumnId) => {
    const next = nextListSort(sort, col)
    persistSort(next.column, next.dir)
  }
  const cells: ReactNode[] = []
  for (const id of cols) {
    if (id === 'name') {
      cells.push(<Cell key="tile" onContextMenu={onMenu} />)
    }
    cells.push(
      <HeaderCell
        key={id}
        id={id}
        label={headerLabel(id, t)}
        sortColumn={sort.column}
        sortDir={sort.dir}
        onSort={onSort}
        onMenu={onMenu}
      />,
    )
  }
  return cells
}

function rowGroupKey(by: GroupBy, row: ListRow): string {
  if (by === 'category') {
    return row.details?.category ?? ''
  }
  if (by === 'source') {
    return row.source
  }
  if (by === 'tag') {
    return firstTag(row.tags)
  }
  return ''
}

export function ModList({ profile, mods }: { profile: Profile; mods: Mod[] }) {
  const { t, i18n } = useLingui()
  const narrow = useMediaQuery(compactQuery)
  const listColumns = useSettings((s) => s.listColumns)
  const listSortColumn = useSettings((s) => s.listSortColumn)
  const listSortDir = useSettings((s) => s.listSortDir)
  const groupBy = sanitizeListGroupBy(useSettings((s) => s.listGroupBy))
  const visible = sanitizeListColumns(listColumns)
  const cols = visibleListColumns(listColumns, narrow)
  const sort = sanitizeListSort(listSortColumn ?? '', listSortDir ?? '')
  const [menu, setMenu] = useState<{ top: number; left: number } | null>(null)
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})
  const byId = useNexusDetails((s) => s.byId)

  const tagHint = t`A mod with several tags appears under its first tag.`

  useEffect(() => {
    primeDetails(mods.map((m) => nexusIdOf(profile, m)).filter((id) => id > 0)).catch(
      reportUnexpected,
    )
  }, [mods, profile])

  const groups = groupSorted(
    mods.map((m) => toListRow(m, profile, byId, t)),
    groupBy,
    (row) => rowGroupKey(groupBy, row),
    (a, b) => compareListRows(a, b, sort),
  )
  const orderedIds = groups.flatMap((g) => g.items.map((r) => modId(r.mod)))
  const onMenu = (e: MouseEvent) => setMenu(columnMenuFromEvent(e))
  const grid = listGridColumns(cols)
  const emptyLabel = emptyGroupLabel(groupBy, {
    category: t`Uncategorised`,
    source: t`Unknown source`,
    tag: t`Untagged`,
  })

  return (
    <Box sx={{ minWidth: 0, minHeight: 0, overflowY: 'auto' }}>
      <Table
        aria-label={t`Mods`}
        stickyHeader={true}
        sx={{
          display: 'block',
          '& thead, & tbody': { display: 'block' },
        }}
      >
        <TableHead>
          <TableRow
            sx={{
              display: 'grid',
              gridTemplateColumns: grid,
              gap: '10px',
              alignItems: 'center',
              px: 2,
              height: 30,
              position: 'sticky',
              top: 0,
              zIndex: 1,
              bgcolor: 'rgba(25,25,30,0.9)',
              ...heading,
              borderBottom: '1px solid rgba(255,255,255,0.08)',
            }}
          >
            <HeaderCells cols={cols} sort={sort} t={t} onMenu={onMenu} />
          </TableRow>
        </TableHead>
        <TableBody>
          {groups.map((group) => {
            const open = collapsed[group.key] !== true
            return (
              <Box key={group.key || 'none'} component="div">
                {groupBy === 'none' ? null : (
                  <TableRow sx={{ display: 'block' }}>
                    <TableCell sx={{ p: 0, border: 0, display: 'block' }}>
                      <ModsGroupHeader
                        label={group.key || emptyLabel}
                        count={group.items.length}
                        open={open}
                        onToggle={() => setCollapsed((cur) => ({ ...cur, [group.key]: open }))}
                        {...(groupBy === 'tag' ? { hint: tagHint } : {})}
                      />
                    </TableCell>
                  </TableRow>
                )}
                {open
                  ? group.items.map((r, i) => (
                      <ModRow
                        key={modId(r.mod)}
                        row={r}
                        striped={i % 2 === 1}
                        cols={cols}
                        locale={i18n.locale}
                        orderedIds={orderedIds}
                        profile={profile}
                      />
                    ))
                  : null}
              </Box>
            )
          })}
        </TableBody>
      </Table>
      <ListColumnMenu
        anchor={menu}
        visible={visible}
        onToggle={(id) => persistColumns(toggleListColumn(visible, id))}
        onReset={() => persistColumns([...DEFAULT_VISIBLE_LIST_COLUMNS])}
        onClose={() => setMenu(null)}
      />
    </Box>
  )
}
