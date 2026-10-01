import { i18n, type MessageDescriptor } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Divider,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  TableCell,
  type TableCellProps,
} from '@mui/material'
import { ArrowDown, ArrowUp, Check, RotateCcw } from 'lucide-react'
import type { DragEvent, MouseEvent, ReactNode } from 'react'
import { SetListSort } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useSettings } from '../settings/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import {
  LIST_COLUMN_IDS,
  type ListColumnId,
  LOCKED_LIST_COLUMNS,
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
}

function columnLabel(id: ListColumnId): string {
  return i18n._(COLUMN_LABELS[id])
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
  onDropColumn,
}: {
  id: ListColumnId
  label: string
  sortColumn: ListColumnId
  sortDir: 'asc' | 'desc'
  onSort: (id: ListColumnId) => void
  onMenu: (e: MouseEvent) => void
  onDropColumn: (from: ListColumnId, to: ListColumnId) => void
}) {
  const active = id !== 'on' && sortColumn === id
  const onDragStart = (e: DragEvent) => {
    e.dataTransfer.setData('text/plain', id)
    e.dataTransfer.effectAllowed = 'move'
  }
  const onDragOver = (e: DragEvent) => {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
  }
  const onDrop = (e: DragEvent) => {
    e.preventDefault()
    const from = e.dataTransfer.getData('text/plain')
    if (from !== '' && from !== id) {
      onDropColumn(from as ListColumnId, id)
    }
  }
  return (
    <Cell
      title={label}
      draggable={true}
      onDragStart={onDragStart}
      onDragOver={onDragOver}
      onDrop={onDrop}
      onClick={id === 'on' ? undefined : () => onSort(id)}
      onContextMenu={onMenu}
      sx={{
        ...ellipsis,
        cursor: id === 'on' ? 'grab' : 'pointer',
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

function persistSort(column: ListColumnId, dir: 'asc' | 'desc') {
  useSettings.setState({ listSortColumn: column, listSortDir: dir })
  SetListSort(column, dir).catch(reportUnexpected)
}

function HeaderCells({
  cols,
  sort,
  onMenu,
  onDropColumn,
}: {
  cols: readonly ListColumnId[]
  sort: ReturnType<typeof sanitizeListSort>
  onMenu: (e: MouseEvent) => void
  onDropColumn: (from: ListColumnId, to: ListColumnId) => void
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
        label={columnLabel(id)}
        sortColumn={sort.column}
        sortDir={sort.dir}
        onSort={onSort}
        onMenu={onMenu}
        onDropColumn={onDropColumn}
      />,
    )
  }
  return cells
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
      {LIST_COLUMN_IDS.map((id) => {
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
            <ListItemIcon>{shown ? <Check size={16} aria-hidden={true} /> : null}</ListItemIcon>
            <ListItemText>{columnLabel(id)}</ListItemText>
          </MenuItem>
        )
      })}
      <Divider />
      <MenuItem
        onClick={() => {
          onReset()
          onClose()
        }}
      >
        <ListItemIcon>
          <RotateCcw size={16} aria-hidden={true} />
        </ListItemIcon>
        <ListItemText>{t`Reset to default columns`}</ListItemText>
      </MenuItem>
    </Menu>
  )
}

export { HeaderCells, ListColumnMenu }
