import { useLingui } from '@lingui/react/macro'
import { Divider, ListItemIcon, ListItemText, Menu, MenuItem } from '@mui/material'
import { Check, RotateCcw } from 'lucide-react'
import { LIST_COLUMN_IDS, type ListColumnId, LOCKED_LIST_COLUMNS } from './listColumns.ts'

function columnLabel(id: ListColumnId, t: ReturnType<typeof useLingui>['t']): string {
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

export function ListColumnMenu({
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
            <ListItemText>{columnLabel(id, t)}</ListItemText>
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
