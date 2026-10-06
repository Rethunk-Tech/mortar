import { useLingui } from '@lingui/react/macro'
import { Box, Table, TableBody, TableHead, TableRow } from '@mui/material'
import { type MouseEvent, type ReactNode, type Ref, useCallback, useRef } from 'react'
import { showModId, useDetail } from './detail.ts'
import type { sanitizeListGroupBy } from './group.ts'
import { HeaderCells, ListColumnMenu } from './ListColumnMenu.tsx'
import {
  DEFAULT_VISIBLE_LIST_COLUMNS,
  type ListColumnId,
  type ListRow,
  persistColumns,
  type sanitizeListSort,
  toggleListColumn,
} from './listColumns.ts'
import { modId } from './lookup.ts'
import { GroupHeaderRow } from './ModsGroupHeader.tsx'
import { OverlayListRow } from './OverlayRow.tsx'
import { heading } from './paper.ts'
import {
  focusModAt,
  LIST_ROW_PX,
  listRowId,
  orderedModIds,
  stepId,
  useModReveal,
  useModTypeahead,
  useModVirtual,
  type VirtualRow,
} from './virtualRows.ts'

function ListShell({
  grid,
  cols,
  sort,
  onMenu,
  onPreview,
  onCommit,
  onCancel,
  label,
  parentRef,
  total,
  menu,
  visible,
  setMenu,
  children,
}: {
  grid: string
  cols: readonly ListColumnId[]
  sort: ReturnType<typeof sanitizeListSort>
  onMenu: (e: MouseEvent) => void
  onPreview: (order: ListColumnId[]) => void
  onCommit: () => void
  onCancel: () => void
  label: string
  parentRef: Ref<HTMLDivElement>
  total: number
  menu: { top: number; left: number } | null
  visible: ListColumnId[]
  setMenu: (menu: { top: number; left: number } | null) => void
  children: ReactNode
}) {
  return (
    // Header and body are separate tables so the header stays put while rows scroll; ARIA joins them into one
    // table so each cell is read with its column.
    <Box
      role="table"
      aria-label={label}
      sx={{ minWidth: 0, minHeight: 0, height: '100%', display: 'flex', flexDirection: 'column' }}
    >
      <Table
        role="presentation"
        sx={{
          display: 'block',
          flex: '0 0 auto',
          '& thead': { display: 'block' },
        }}
      >
        <TableHead role="rowgroup">
          <TableRow
            role="row"
            sx={{
              display: 'grid',
              gridTemplateColumns: grid,
              gap: '10px',
              alignItems: 'center',
              px: 2,
              height: 30,
              zIndex: 1,
              bgcolor: 'var(--mortar-console-90)',
              ...heading,
              borderBottom: '1px solid var(--mortar-hairline-muted)',
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
      </Table>
      <Box
        ref={parentRef}
        tabIndex={0}
        onKeyDown={(e) => {
          if (e.target === e.currentTarget && (e.key === 'ArrowDown' || e.key === 'ArrowUp')) {
            e.preventDefault()
            e.currentTarget.querySelector<HTMLElement>('[data-mod-row]')?.focus()
          }
        }}
        sx={{
          flex: 1,
          minHeight: 0,
          overflowY: 'auto',
          pb: 1.5,
          '&:focus-visible': { outlineOffset: -2 },
        }}
      >
        <Table role="presentation" sx={{ display: 'block', '& tbody': { display: 'block' } }}>
          <TableBody role="rowgroup" sx={{ display: 'block', position: 'relative', height: total }}>
            {children}
          </TableBody>
        </Table>
      </Box>
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

function ListSlot({
  item,
  headingFor,
  collapsed,
  gameId,
  setCollapsed,
  groupBy,
  tagHint,
  groups,
  renderRow,
  onArrow,
}: {
  item: VirtualRow<ListRow>
  headingFor: (key: string) => string
  collapsed: Record<string, boolean>
  gameId: string
  setCollapsed: (fn: (cur: Record<string, boolean>) => Record<string, boolean>) => void
  groupBy: ReturnType<typeof sanitizeListGroupBy>
  tagHint: string
  groups: readonly { key: string; items: readonly ListRow[] }[]
  renderRow: (
    row: ListRow,
    striped: boolean,
    onArrow: (id: string, dir: -1 | 1) => void,
  ) => ReactNode
  onArrow: (id: string, dir: -1 | 1) => void
}) {
  if (item.kind === 'header') {
    return (
      <Box role="row" sx={{ display: 'contents' }}>
        <Box role="cell" sx={{ display: 'contents' }}>
          <GroupHeaderRow
            groupKey={item.groupKey}
            count={item.count}
            label={headingFor(item.groupKey)}
            collapsed={collapsed}
            gameId={gameId}
            setCollapsed={setCollapsed}
            groupBy={groupBy}
            tagHint={tagHint}
            groups={groups}
          />
        </Box>
      </Box>
    )
  }
  if (item.kind === 'row') {
    return renderRow(item.item, item.stripe, onArrow)
  }
  if (item.kind === 'overlay') {
    return <OverlayListRow overlay={item.overlay} />
  }
  return null
}

export function ModListTable({
  grid,
  cols,
  sort,
  onMenu,
  onPreview,
  onCommit,
  onCancel,
  groups,
  items,
  groupBy,
  headingFor,
  tagHint,
  collapsed,
  gameId,
  setCollapsed,
  visible,
  menu,
  setMenu,
  renderRow,
}: {
  grid: string
  cols: readonly ListColumnId[]
  sort: ReturnType<typeof sanitizeListSort>
  onMenu: (e: MouseEvent) => void
  onPreview: (order: ListColumnId[]) => void
  onCommit: () => void
  onCancel: () => void
  groups: { key: string; items: ListRow[] }[]
  items: VirtualRow<ListRow>[]
  groupBy: ReturnType<typeof sanitizeListGroupBy>
  headingFor: (key: string) => string
  tagHint: string
  collapsed: Record<string, boolean>
  gameId: string
  setCollapsed: (fn: (cur: Record<string, boolean>) => Record<string, boolean>) => void
  visible: ListColumnId[]
  menu: { top: number; left: number } | null
  setMenu: (menu: { top: number; left: number } | null) => void
  renderRow: (
    row: ListRow,
    striped: boolean,
    onArrow: (id: string, dir: -1 | 1) => void,
  ) => ReactNode
}) {
  const { t } = useLingui()
  const detailId = useDetail((s) => s.detailId)
  const { parentRef, virtualizer } = useModVirtual(items, LIST_ROW_PX)
  const navIds = orderedModIds(items, (row) => modId(row.mod))
  useModTypeahead({
    items,
    nameOf: (row) => row.mod.name,
    idOf: listRowId,
    virtualizer,
    parentRef,
  })
  useModReveal({
    detailId,
    groups,
    items,
    idOf: listRowId,
    collapsed,
    setCollapsed,
    gameId,
    virtualizer,
  })
  const arrow = useRef<(id: string, dir: -1 | 1) => void>(() => undefined)
  arrow.current = (id, dir) => {
    const next = stepId(navIds, id, dir)
    if (next && next !== id) {
      focusModAt({ items, idOf: listRowId, virtualizer, parentRef }, next)
      showModId(next)
    }
  }
  // Stable, so memoised rows keep their props across renders while still stepping through the current items.
  const onArrow = useCallback((id: string, dir: -1 | 1) => arrow.current(id, dir), [])
  return (
    <ListShell
      grid={grid}
      cols={cols}
      sort={sort}
      onMenu={onMenu}
      onPreview={onPreview}
      onCommit={onCommit}
      onCancel={onCancel}
      label={t`Mods`}
      parentRef={parentRef}
      total={virtualizer.getTotalSize()}
      menu={menu}
      visible={visible}
      setMenu={setMenu}
    >
      {virtualizer.getVirtualItems().map((vi) => {
        const item = items[vi.index]
        if (!item) {
          return null
        }
        return (
          <Box
            key={item.key}
            data-index={vi.index}
            sx={{
              position: 'absolute',
              top: 0,
              left: 0,
              width: '100%',
              transform: `translateY(${vi.start}px)`,
            }}
          >
            <ListSlot
              item={item}
              headingFor={headingFor}
              collapsed={collapsed}
              gameId={gameId}
              setCollapsed={setCollapsed}
              groupBy={groupBy}
              tagHint={tagHint}
              groups={groups}
              renderRow={renderRow}
              onArrow={onArrow}
            />
          </Box>
        )
      })}
    </ListShell>
  )
}
