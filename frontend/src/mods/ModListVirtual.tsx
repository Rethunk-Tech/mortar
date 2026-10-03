import { useLingui } from '@lingui/react/macro'
import { Box, Table, TableBody, TableHead, TableRow } from '@mui/material'
import { type MouseEvent, type ReactNode, type Ref, useLayoutEffect, useRef } from 'react'
import { reportUnexpected } from '../toasts/report.ts'
import { useDetail } from './detail.ts'
import { type sanitizeListGroupBy, toggleCollapsed } from './group.ts'
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
import { ModsGroupHeader } from './ModsGroupHeader.tsx'
import { heading } from './paper.ts'
import { useMods } from './store.ts'
import { setGroupEnabled } from './storeEntries.ts'
import {
  groupKeyHolding,
  LIST_ROW_PX,
  neighborId,
  orderedModIds,
  useModTypeahead,
  useModVirtual,
  type VirtualRow,
  virtualIndexOf,
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
    <Box
      sx={{ minWidth: 0, minHeight: 0, height: '100%', display: 'flex', flexDirection: 'column' }}
    >
      <Table
        aria-label={label}
        sx={{
          display: 'block',
          flex: '0 0 auto',
          '& thead': { display: 'block' },
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
        sx={{ flex: 1, minHeight: 0, overflowY: 'auto', pb: 1.5, outline: 'none' }}
      >
        <Table sx={{ display: 'block', '& tbody': { display: 'block' } }}>
          <TableBody sx={{ display: 'block', position: 'relative', height: total }}>
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
      <ModsGroupHeader
        label={headingFor(item.groupKey)}
        count={item.count}
        open={collapsed[item.groupKey] !== true}
        onToggle={() =>
          setCollapsed((cur) =>
            toggleCollapsed(gameId, cur, item.groupKey, collapsed[item.groupKey] !== true),
          )
        }
        {...(groupBy === 'tag' ? { hint: tagHint } : {})}
        {...(groupBy === 'group' && item.groupKey !== ''
          ? {
              enabled:
                groups.find((g) => g.key === item.groupKey)?.items.every((r) => r.mod.enabled) ===
                true,
              onEnabled: (on: boolean) => {
                setGroupEnabled(item.groupKey, on)
                  .then(() => useMods.getState().load())
                  .catch(reportUnexpected)
              },
            }
          : {})}
      />
    )
  }
  if (item.kind === 'row') {
    return renderRow(item.item, item.stripe, onArrow)
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
  const lastReveal = useRef('')
  const navIds = orderedModIds(items, (row) => modId(row.mod))
  useModTypeahead({
    items,
    nameOf: (row) => row.mod.name,
    idOf: (row) => modId(row.mod),
    virtualizer,
    parentRef,
  })
  useLayoutEffect(() => {
    if (!detailId) {
      return
    }
    const held = groupKeyHolding(groups, (row) => modId(row.mod) === detailId)
    if (held !== undefined && collapsed[held] === true) {
      setCollapsed((cur) => toggleCollapsed(gameId, cur, held, false))
      return
    }
    const idx = virtualIndexOf(items, detailId, (row) => modId(row.mod))
    const token = `${detailId}:${idx}`
    if (lastReveal.current === token || idx < 0) {
      return
    }
    lastReveal.current = token
    virtualizer.scrollToIndex(idx, { align: 'auto' })
  }, [collapsed, detailId, gameId, groups, items, setCollapsed, virtualizer])
  const onArrow = (id: string, dir: -1 | 1) => {
    const next = neighborId(navIds, id, dir)
    if (!next) {
      return
    }
    const idx = virtualIndexOf(items, next, (row) => modId(row.mod))
    if (idx >= 0) {
      virtualizer.scrollToIndex(idx, { align: 'auto' })
    }
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        parentRef.current
          ?.querySelector<HTMLElement>(`[data-mod-id="${CSS.escape(next)}"]`)
          ?.focus()
      })
    })
  }
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
