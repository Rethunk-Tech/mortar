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
import { useLingui } from '@lingui/react/macro'
import {
  Box,
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
import { SetListSort } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useSettings } from '../settings/store.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { MenuRule } from '../shell/TitleMenu.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { columnLabel } from './columnLabel.ts'
import { useColumnAvailable } from './contributedColumns.ts'
import {
  LIST_COLUMN_GROUPS,
  type ListColumnId,
  LOCKED_LIST_COLUMNS,
  moveListColumn,
  nextListSort,
  type sanitizeListSort,
} from './listColumns.ts'

const DRAG_TINT = 0.16
const ARIA_SORT = { asc: 'ascending', desc: 'descending' } as const

function headerCursor(id: ListColumnId, isDragging: boolean): string {
  if (isDragging) {
    return 'grabbing'
  }
  return id === 'on' ? 'grab' : 'pointer'
}

const ellipsis = { overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } as const
const visuallyHidden = {
  position: 'absolute',
  width: 1,
  height: 1,
  overflow: 'hidden',
  clip: 'rect(0 0 0 0)',
  whiteSpace: 'nowrap',
} as const
const cellBase = { p: 0, border: 0, fontSize: 'inherit', color: 'inherit' } as const

// The role comes last: dnd-kit's sortable attributes would otherwise make each header a button.
function Cell({ sx, ...props }: TableCellProps) {
  return <TableCell {...props} role="columnheader" sx={{ ...cellBase, ...sx }} />
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
      aria-sort={active ? ARIA_SORT[sortDir] : undefined}
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
  const { t } = useLingui()
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
      cells.push(
        <Cell key="tile" onContextMenu={onMenu}>
          <Box component="span" sx={visuallyHidden}>
            {t`Icon`}
          </Box>
        </Cell>,
      )
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
      // The header row is a <tr>, which cannot hold dnd-kit's hidden screen-reader text; it goes to the body instead.
      accessibility={{ container: document.body }}
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
  const available = useColumnAvailable()
  return (
    <Menu
      open={anchor !== null}
      onClose={onClose}
      anchorReference="anchorPosition"
      anchorPosition={anchor ?? undefined}
    >
      {LIST_COLUMN_GROUPS.flatMap((group, index) => [
        index > 0 ? <MenuRule key={`divider-${group[0]}`} /> : null,
        ...group.filter(available).map((id) => {
          const locked = LOCKED_LIST_COLUMNS.includes(id)
          const shown = visible.includes(id)
          return (
            <MenuItem
              key={id}
              role="menuitemcheckbox"
              aria-checked={shown}
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
      <MenuRule />
      <MenuAction
        icon={<RotateCcw size={16} aria-hidden={true} />}
        label={t`Reset to default columns`}
        onClick={() => {
          onReset()
          onClose()
        }}
      />
    </Menu>
  )
}

export { HeaderCells, ListColumnMenu }
