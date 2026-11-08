import { useLingui } from '@lingui/react/macro'
import { Box, Table, TableBody, TableHead, TableRow } from '@mui/material'
import { sameValue } from '@rethunk/react-load-hooks'
import { type MouseEvent, memo, type ReactNode, type Ref } from 'react'
import { space } from '../theme/density.ts'
import type { sanitizeListGroupBy } from './group.ts'
import { HeaderCells, ListColumnMenu } from './ListColumnMenu.tsx'
import {
  DEFAULT_VISIBLE_LIST_COLUMNS,
  type ListColumnId,
  type ListRow,
  type sanitizeListSort,
  toggleListColumn,
} from './listColumns.ts'
import { persistColumns } from './listPrefs.ts'
import { GroupHeaderRow, StickyGroupSlot } from './ModsGroupHeader.tsx'
import { OverlayListRow } from './OverlayRow.tsx'
import { heading } from './paper.ts'
import {
  LIST_ROW_PX,
  listRowId,
  type ModGroupChrome,
  RevealOpen,
  stuckHeader,
  useModStep,
  useModTypeahead,
  useModVirtual,
  type VirtualRow,
} from './virtualRows.ts'

interface HeaderProps {
  grid: string
  cols: readonly ListColumnId[]
  sort: ReturnType<typeof sanitizeListSort>
  onMenu: (e: MouseEvent) => void
  onPreview: (order: ListColumnId[]) => void
  onCommit: () => void
  onCancel: () => void
}

function ListHeaderView({ grid, cols, sort, onMenu, onPreview, onCommit, onCancel }: HeaderProps) {
  return (
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
            gap: space.gap,
            alignItems: 'center',
            px: space.gutter,
            height: `calc(${space.row} - 6px)`,
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
  )
}

// The list re-renders for every answer that changes a row; the header redraws only when its columns or sort change.
// Its handlers capture nothing beyond those, so a skipped render never leaves them stale.
const ListHeader = memo(
  ListHeaderView,
  (a, b) => a.grid === b.grid && a.cols.join() === b.cols.join() && sameValue(a.sort, b.sort),
)

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
  stuck,
  children,
}: HeaderProps & {
  label: string
  parentRef: Ref<HTMLDivElement>
  total: number
  menu: { top: number; left: number } | null
  visible: ListColumnId[]
  setMenu: (menu: { top: number; left: number } | null) => void
  stuck: ReactNode
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
      <ListHeader
        grid={grid}
        cols={cols}
        sort={sort}
        onMenu={onMenu}
        onPreview={onPreview}
        onCommit={onCommit}
        onCancel={onCancel}
      />
      <Box
        ref={parentRef}
        role="rowgroup"
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
          pb: space.pad,
          '&:focus-visible': { outlineOffset: -2 },
        }}
      >
        {stuck}
        <Table role="presentation" sx={{ display: 'block', '& tbody': { display: 'block' } }}>
          <TableBody
            role="presentation"
            sx={{ display: 'block', position: 'relative', height: total }}
          >
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
  group,
  renderRow,
  onArrow,
}: {
  item: VirtualRow<ListRow>
  headingFor: (key: string) => string
  group: ModGroupChrome
  renderRow: (row: ListRow, onArrow: (id: string, dir: -1 | 1) => void) => ReactNode
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
            {...group}
          />
        </Box>
      </Box>
    )
  }
  if (item.kind === 'row') {
    return renderRow(item.item, onArrow)
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
  collapsed,
  gameId,
  setCollapsed,
  visible,
  menu,
  setMenu,
  renderRow,
}: HeaderProps & {
  groups: { key: string; items: ListRow[] }[]
  items: VirtualRow<ListRow>[]
  groupBy: ReturnType<typeof sanitizeListGroupBy>
  headingFor: (key: string) => string
  collapsed: Record<string, boolean>
  gameId: string
  setCollapsed: (fn: (cur: Record<string, boolean>) => Record<string, boolean>) => void
  visible: ListColumnId[]
  menu: { top: number; left: number } | null
  setMenu: (menu: { top: number; left: number } | null) => void
  renderRow: (row: ListRow, onArrow: (id: string, dir: -1 | 1) => void) => ReactNode
}) {
  const { t } = useLingui()
  const { parentRef, virtualizer } = useModVirtual(items, LIST_ROW_PX)
  useModTypeahead({
    items,
    nameOf: (row) => row.mod.name,
    idOf: listRowId,
    virtualizer,
    parentRef,
  })
  const onArrow = useModStep(items, virtualizer, parentRef)
  const group: ModGroupChrome = { collapsed, gameId, setCollapsed, groupBy, groups }
  const stuck = stuckHeader(items, virtualizer.measurementsCache, virtualizer.scrollOffset ?? 0)
  return (
    <>
      <RevealOpen
        groups={groups}
        items={items}
        idOf={listRowId}
        collapsed={collapsed}
        setCollapsed={setCollapsed}
        gameId={gameId}
        virtualizer={virtualizer}
      />
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
        stuck={
          stuck ? (
            <StickyGroupSlot shift={stuck.shift}>
              <GroupHeaderRow
                groupKey={stuck.row.groupKey}
                count={stuck.row.count}
                label={headingFor(stuck.row.groupKey)}
                {...group}
              />
            </StickyGroupSlot>
          ) : null
        }
      >
        {virtualizer.getVirtualItems().map((vi) => {
          const item = items[vi.index]
          if (!item) {
            return null
          }
          return (
            // A plain element: the slot re-renders with the list, and a styled Box costs several components each.
            <div
              key={item.key}
              data-index={vi.index}
              // The stripe lives on the slot, so a mod moving group restripes the slots, not every memoised row.
              data-striped={item.kind === 'row' && item.stripe ? '' : undefined}
              style={{
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
                group={group}
                renderRow={renderRow}
                onArrow={onArrow}
              />
            </div>
          )
        })}
      </ListShell>
    </>
  )
}
