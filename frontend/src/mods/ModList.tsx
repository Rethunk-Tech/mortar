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
import { Pin } from 'lucide-react'
import { type MouseEvent, type ReactNode, useEffect, useState } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { compactQuery } from '../game/compact.ts'
import { When } from '../i18n/When.tsx'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useCustomCategories } from './customCategories.ts'
import { useDetail } from './detail.ts'
import {
  customCategoryById,
  emptyGroupLabel,
  groupHeading,
  groupSorted,
  installedNames,
  loadCollapsed,
  rowGroupKey,
  sanitizeListGroupBy,
  toggleCollapsed,
} from './group.ts'
import { HeaderCells, ListColumnMenu } from './ListColumnMenu.tsx'
import {
  columnMenuFromEvent,
  compareListRows,
  DEFAULT_VISIBLE_LIST_COLUMNS,
  type ListColumnId,
  type ListRow,
  listGridColumns,
  persistColumns,
  sanitizeListColumns,
  sanitizeListSort,
  toggleListColumn,
  visibleListColumns,
} from './listColumns.ts'
import { toListRow } from './listRows.ts'
import { modId, modStatusProblem, nexusIdOf, updateFor } from './lookup.ts'
import { ModsGroupHeader } from './ModsGroupHeader.tsx'
import { contextMenuProps } from './menu.ts'
import { primeDetails, useNexusDetails, useNexusFresh } from './nexusDetails.ts'
import { formatCount, isNewer } from './nexusFormat.ts'
import { heading } from './paper.ts'
import {
  LastRunBadge,
  LetterTile,
  ModSwitch,
  NexusGoneBadge,
  PinBadge,
  ProblemBadge,
  UpdateBadge,
} from './parts.tsx'
import { useSelection } from './selection.ts'
import { useMods } from './store.ts'
import { useUpdates } from './updates.ts'

const SELECTED_ALPHA = 0.14
const ellipsis = { overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } as const

const cellBase = { p: 0, border: 0, fontSize: 'inherit', color: 'inherit' } as const

function Cell({ sx, ...props }: TableCellProps) {
  return <TableCell {...props} sx={{ ...cellBase, ...sx }} />
}

function dash(value: string) {
  return value || '—'
}

function ValueCell({ text, title, accent }: { text: ReactNode; title?: string; accent?: boolean }) {
  return (
    <Cell
      title={title ?? (typeof text === 'string' ? text : undefined)}
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
    case 'author':
    case 'source':
    case 'category':
    case 'updated':
    case 'installed':
    case 'notes': {
      const notes = [row.note, row.tags.join(', ')].filter((part) => part !== '').join(' · ')
      if (id === 'updated') {
        return <ValueCell key={id} text={<When value={page?.updated ?? ''} />} />
      }
      if (id === 'installed') {
        return <ValueCell key={id} text={<When value={row.added} />} />
      }
      const text = {
        uniqueId: dash(m.uniqueId),
        author: dash(m.author),
        source: dash(row.source),
        category: dash(row.categoryLabel),
        notes: dash(notes),
      }[id]
      return <ValueCell key={id} text={text} />
    }
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
            <NexusGoneBadge mod={m} />
            <UpdateBadge mod={m} />
            <Box component="span" title={row.status} sx={ellipsis}>
              {row.status}
            </Box>
          </Box>
        </Cell>
      )
    case 'lastRun':
      return (
        <Cell key="lastRun">
          <LastRunBadge mod={m} />
        </Cell>
      )
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
  const setEnabled = useMods((s) => s.setEnabled)
  const askRemove = useMods((s) => s.askRemove)
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
      data-mod-row="true"
      tabIndex={orderedIds[0] === rowId ? 0 : -1}
      {...menu}
      onKeyDown={(e) => {
        if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
          e.preventDefault()
          const rows = [
            ...(e.currentTarget.parentElement?.querySelectorAll<HTMLElement>(
              '[data-mod-row="true"]',
            ) ?? []),
          ]
          const index = rows.indexOf(e.currentTarget)
          rows[(index + (e.key === 'ArrowDown' ? 1 : -1) + rows.length) % rows.length]?.focus()
          return
        }
        if (e.key === ' ') {
          e.preventDefault()
          setEnabled(m, !m.enabled).catch(reportUnexpected)
          return
        }
        if (e.key === 'Delete') {
          e.preventDefault()
          askRemove(m)
          return
        }
        menu.onKeyDown(e)
      }}
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

function ModListTable({
  grid,
  cols,
  sort,
  onMenu,
  onPreview,
  onCommit,
  onCancel,
  groups,
  groupBy,
  headingFor,
  tagHint,
  collapsed,
  gameId,
  setCollapsed,
  locale,
  orderedIds,
  profile,
  visible,
  menu,
  setMenu,
}: {
  grid: string
  cols: readonly ListColumnId[]
  sort: ReturnType<typeof sanitizeListSort>
  onMenu: (e: MouseEvent) => void
  onPreview: (order: ListColumnId[]) => void
  onCommit: () => void
  onCancel: () => void
  groups: { key: string; items: ListRow[] }[]
  groupBy: ReturnType<typeof sanitizeListGroupBy>
  headingFor: (key: string) => string
  tagHint: string
  collapsed: Record<string, boolean>
  gameId: string
  setCollapsed: (fn: (cur: Record<string, boolean>) => Record<string, boolean>) => void
  locale: string
  orderedIds: readonly string[]
  profile: Profile
  visible: ListColumnId[]
  menu: { top: number; left: number } | null
  setMenu: (menu: { top: number; left: number } | null) => void
}) {
  const { t } = useLingui()
  return (
    <Box sx={{ minWidth: 0, minHeight: 0, overflowY: 'auto', pb: 1.5 }}>
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
            <HeaderCells
              cols={cols}
              sort={sort}
              onMenu={onMenu}
              onPreview={onPreview}
              onCommit={onCommit}
              onCancel={onCancel}
            />
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
                        label={headingFor(group.key)}
                        count={group.items.length}
                        open={open}
                        onToggle={() =>
                          setCollapsed((cur) => toggleCollapsed(gameId, cur, group.key, open))
                        }
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
                        locale={locale}
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

export function ModList({ profile, mods }: { profile: Profile; mods: Mod[] }) {
  const { t, i18n } = useLingui()
  const narrow = useMediaQuery(compactQuery)
  const listColumns = useSettings((s) => s.listColumns)
  const listSortColumn = useSettings((s) => s.listSortColumn)
  const listSortDir = useSettings((s) => s.listSortDir)
  const groupBy = sanitizeListGroupBy(useSettings((s) => s.listGroupBy))
  const visible = sanitizeListColumns(listColumns)
  const settled = visibleListColumns(listColumns, narrow)
  // While a header is dragged the table shows this order, so every row moves with it.
  const [preview, setPreview] = useState<ListColumnId[] | null>(null)
  const cols = preview ?? settled
  const sort = sanitizeListSort(listSortColumn ?? '', listSortDir ?? '')
  const [menu, setMenu] = useState<{ top: number; left: number } | null>(null)
  const gameId = useProfiles((s) => s.game?.id) ?? ''
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>(() => loadCollapsed(gameId))
  const byId = useNexusDetails((s) => s.byId)
  const customCategories = useCustomCategories((s) => s.categories)
  const customById = customCategoryById(customCategories)
  const problems = useMods((s) => s.problems)
  const updates = useUpdates((s) => s.updates)

  const tagHint = t`A mod with several tags appears under its first tag.`

  useEffect(() => {
    setCollapsed(loadCollapsed(gameId))
  }, [gameId])

  useEffect(() => {
    primeDetails(mods.map((m) => nexusIdOf(profile, m)).filter((id) => id > 0)).catch(
      reportUnexpected,
    )
  }, [mods, profile])

  const names = installedNames(mods)
  const groups = groupSorted(
    mods.map((m) => toListRow(m, profile, byId, customCategories)),
    groupBy,
    (row) =>
      rowGroupKey(groupBy, row, {
        hasProblem: modStatusProblem(problems, row.mod),
        hasUpdate: Boolean(updateFor(updates, row.mod, profile)),
        names,
        customById,
      }),
    (a, b) => compareListRows(a, b, sort),
  )
  const orderedIds = groups.flatMap((g) => g.items.map((r) => modId(r.mod)))
  const onMenu = (e: MouseEvent) => setMenu(columnMenuFromEvent(e))
  const grid = listGridColumns(cols)
  const emptyLabel = emptyGroupLabel(groupBy, {
    category: t`Uncategorised`,
    source: t`Unknown source`,
    tag: t`Untagged`,
    author: t`Unknown author`,
  })
  const headingFor = (key: string) =>
    groupHeading(groupBy, key, {
      empty: emptyLabel,
      problems: t`Problems`,
      update: t`Update available`,
      enabled: t`Enabled`,
      disabled: t`Disabled`,
      smapi: t`SMAPI mods`,
    })
  const onCommit = () => {
    if (preview) {
      persistColumns([...preview, ...visible.filter((id) => !preview.includes(id))])
    }
    setPreview(null)
  }
  const onCancel = () => setPreview(null)

  return (
    <ModListTable
      grid={grid}
      cols={cols}
      sort={sort}
      onMenu={onMenu}
      onPreview={setPreview}
      onCommit={onCommit}
      onCancel={onCancel}
      groups={groups}
      groupBy={groupBy}
      headingFor={headingFor}
      tagHint={tagHint}
      collapsed={collapsed}
      gameId={gameId}
      setCollapsed={setCollapsed}
      locale={i18n.locale}
      orderedIds={orderedIds}
      profile={profile}
      visible={visible}
      menu={menu}
      setMenu={setMenu}
    />
  )
}
