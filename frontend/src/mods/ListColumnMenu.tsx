import { i18n, type MessageDescriptor } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Divider, ListItemIcon, ListItemText, Menu, MenuItem } from '@mui/material'
import { Check, RotateCcw } from 'lucide-react'
import { LIST_COLUMN_IDS, type ListColumnId, LOCKED_LIST_COLUMNS } from './listColumns.ts'

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
