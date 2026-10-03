import {
  closestCenter,
  DndContext,
  type DragOverEvent,
  DragOverlay,
  KeyboardSensor,
  MeasuringStrategy,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core'
import {
  horizontalListSortingStrategy,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
} from '@dnd-kit/sortable'
import { i18n, type MessageDescriptor } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Divider,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  TableCell,
  type TableCellProps,
} from '@mui/material'
import { alpha } from '@mui/material/styles'
import { ArrowDown, ArrowUp, Check, RotateCcw } from 'lucide-react'
import { type MouseEvent, type ReactNode, useRef, useState } from 'react'
import { SetListSort } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useSettings } from '../settings/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import {
  LIST_COLUMN_GROUPS,
  type ListColumnId,
  LOCKED_LIST_COLUMNS,
  moveListColumn,
  nextListSort,
  type sanitizeListSort,
} from './listColumns.ts'

const COLUMN_LABELS: Record<ListColumnId, MessageDescriptor> = {
  on: msg`On`,
  name: msg`Name`,
  version: msg`Version`,
  latest: msg`Latest on Nexus`,
  uniqueId: msg`UniqueID`,
  author: msg`Author`,
  source: msg`Source`,
  category: msg`Category`,
  endorsements: msg`Endorsements`,
  downloads: msg`Downloads`,
  updated: msg`Updated on Nexus`,
  installed: msg`Installed`,
  needs: msg`Needs`,
  status: msg`Status`,
  notes: msg`Notes and tags`,
  lastRun: msg`Last run`,
  size: msg`Size`,
}

function columnLabel(id: ListColumnId): string {
  return i18n._(COLUMN_LABELS[id])
}

const DRAG_TINT = 0.16

function headerCursor(id: ListColumnId, isDragging: boolean): string {
  if (isDragging) {
    return 'grabbing'
  }
  return id === 'on' ? 'grab' : 'pointer'
}

const ellipsis = { overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } as const
const cellBase = { p: 0, border: 0, fontSize: 'inherit', color: 'inherit' } as const

function Cell({ sx, ...props }: TableCellProps) {
  return <TableCell {...props} sx={{ ...cellBase, ...sx }} />
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
  // No transform: the column order itself changes while dragging, so rows move with the header.
  const { attributes, listeners, setNodeRef, isDragging } = useSortable({ id })
  return (
    <Cell
      ref={setNodeRef}
      title={label}
      {...attributes}
      {...listeners}
      onClick={id === 'on' ? undefined : () => onSort(id)}
      onContextMenu={onMenu}
      sx={(theme) => ({
        ...ellipsis,
        cursor: headerCursor(id, isDragging),
        userSelect: 'none',
        display: 'flex',
        alignItems: 'center',
        gap: 0.5,
        borderRadius: '4px',
        outline: isDragging ? `1px solid ${theme.palette.primary.main}` : 'none',
        bgcolor: isDragging ? alpha(theme.palette.primary.main, DRAG_TINT) : 'transparent',
      })}
    >
      {label}
      {active && sortDir === 'asc' ? <ArrowUp size={12} aria-hidden={true} /> : null}
      {active && sortDir === 'desc' ? <ArrowDown size={12} aria-hidden={true} /> : null}
    </Cell>
  )
}

// The ghost follows the pointer while the real header already sits where it will drop.
function HeaderGhost({ label }: { label: string }) {
  return (
    <Box
      sx={(theme) => ({
        display: 'inline-flex',
        alignItems: 'center',
        height: 30,
        px: 1.5,
        borderRadius: '6px',
        border: `1px solid ${theme.palette.primary.main}`,
        bgcolor: 'var(--mortar-menu-95)',
        color: 'var(--mortar-ink)',
        fontSize: 12,
        fontWeight: 700,
        letterSpacing: '0.06em',
        textTransform: 'uppercase',
        whiteSpace: 'nowrap',
        cursor: 'grabbing',
        boxShadow: '0 6px 18px var(--mortar-overlay-45)',
      })}
    >
      {label}
    </Box>
  )
}

function persistSort(column: ListColumnId, dir: 'asc' | 'desc') {
  useSettings.setState({ listSortColumn: column, listSortDir: dir })
  SetListSort(column, dir).catch(reportUnexpected)
}

function HeaderCells({
  cols,
  sort,
  onMenu,
  onPreview,
  onCommit,
  onCancel,
}: {
  cols: readonly ListColumnId[]
  sort: ReturnType<typeof sanitizeListSort>
  onMenu: (e: MouseEvent) => void
  onPreview: (order: ListColumnId[]) => void
  onCommit: () => void
  onCancel: () => void
}) {
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )
  // The click that ends a drag must not also re-sort by the dragged column.
  const dragged = useRef(false)
  const [lifted, setLifted] = useState<ListColumnId | null>(null)
  const onSort = (col: ListColumnId) => {
    if (dragged.current) {
      return
    }
    const next = nextListSort(sort, col)
    persistSort(next.column, next.dir)
  }
  const onDragOver = ({ active, over }: DragOverEvent) => {
    if (!over || active.id === over.id) {
      return
    }
    const from = cols.indexOf(active.id as ListColumnId)
    const to = cols.indexOf(over.id as ListColumnId)
    if (from >= 0 && to >= 0) {
      onPreview(moveListColumn(cols, from, to))
    }
  }
  const settle = () => {
    setLifted(null)
    setTimeout(() => {
      dragged.current = false
    }, 0)
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
        label={columnLabel(id)}
        sortColumn={sort.column}
        sortDir={sort.dir}
        onSort={onSort}
        onMenu={onMenu}
      />,
    )
  }
  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCenter}
      measuring={{ droppable: { strategy: MeasuringStrategy.Always } }}
      onDragStart={({ active }) => {
        dragged.current = true
        setLifted(active.id as ListColumnId)
      }}
      onDragOver={onDragOver}
      onDragEnd={() => {
        onCommit()
        settle()
      }}
      onDragCancel={() => {
        onCancel()
        settle()
      }}
    >
      <SortableContext items={[...cols]} strategy={horizontalListSortingStrategy}>
        {cells}
      </SortableContext>
      <DragOverlay dropAnimation={null}>
        {lifted ? <HeaderGhost label={columnLabel(lifted)} /> : null}
      </DragOverlay>
    </DndContext>
  )
}

function ListColumnMenu({
  anchor,
  visible,
  onToggle,
  onReset,
  onClose,
}: {
  anchor: { top: number; left: number } | null
  visible: readonly ListColumnId[]
  onToggle: (id: ListColumnId) => void
  onReset: () => void
  onClose: () => void
}) {
  const { t } = useLingui()
  return (
    <Menu
      open={anchor !== null}
      onClose={onClose}
      anchorReference="anchorPosition"
      anchorPosition={anchor ?? undefined}
    >
      {LIST_COLUMN_GROUPS.flatMap((group, index) => [
        index > 0 ? <Divider key={`divider-${group[0]}`} /> : null,
        ...group.map((id) => {
          const locked = LOCKED_LIST_COLUMNS.includes(id)
          const shown = visible.includes(id)
          return (
            <MenuItem
              key={id}
              disabled={locked}
              onClick={() => {
                onToggle(id)
              }}
            >
              <ListItemIcon sx={{ color: 'inherit' }}>
                {shown ? <Check size={16} aria-hidden={true} /> : null}
              </ListItemIcon>
              <ListItemText>{columnLabel(id)}</ListItemText>
            </MenuItem>
          )
        }),
      ])}
      <Divider />
      <MenuItem
        onClick={() => {
          onReset()
          onClose()
        }}
      >
        <ListItemIcon sx={{ color: 'inherit' }}>
          <RotateCcw size={16} aria-hidden={true} />
        </ListItemIcon>
        <ListItemText>{t`Reset to default columns`}</ListItemText>
      </MenuItem>
    </Menu>
  )
}

export { columnLabel, HeaderCells, ListColumnMenu }
