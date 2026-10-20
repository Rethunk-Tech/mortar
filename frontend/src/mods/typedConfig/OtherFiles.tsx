import { useLingui } from '@lingui/react/macro'
import { Chip, ListItemButton, ListItemText } from '@mui/material'
import { memo } from 'react'
import { space } from '../../theme/density.ts'
import { GroupHeaderRow } from '../ModsGroupHeader.tsx'
import type { useModGroups } from '../useModGroups.ts'
import { fileSelection } from './configList.ts'

// Memoised: a game can have dozens of these files, and choosing one should redraw two rows, not all of them.
const OtherFileRow = memo(function OtherFileRowView({
  name,
  changed,
  selected,
  onChoose,
}: {
  name: string
  changed: boolean
  selected: boolean
  onChoose: (selection: string) => void
}) {
  const { t } = useLingui()
  return (
    <ListItemButton
      selected={selected}
      onClick={() => onChoose(fileSelection(name))}
      sx={{ minHeight: space.row }}
    >
      <ListItemText primary={name} slotProps={{ primary: { noWrap: true } }} />
      {changed ? <Chip size="small" variant="outlined" label={t`Changed`} /> : null}
    </ListItemButton>
  )
})

// Shares the mod groups' collapse state under a key no group name can take.
const OTHER_GROUP_KEY = '\u0000loader-and-other'

export function OtherFiles({
  view,
  other,
  selection,
  onChoose,
}: {
  view: Pick<ReturnType<typeof useModGroups>, 'collapsed' | 'gameId' | 'setCollapsed'>
  other: { name: string; changed?: boolean }[]
  selection: string | undefined
  onChoose: (selection: string) => void
}) {
  const { t } = useLingui()
  return (
    <>
      <GroupHeaderRow
        groupKey={OTHER_GROUP_KEY}
        count={other.length}
        label={t`Loader and other`}
        collapsed={view.collapsed}
        gameId={view.gameId}
        setCollapsed={view.setCollapsed}
        groupBy="none"
        tagHint=""
        groups={[]}
      />
      {view.collapsed[OTHER_GROUP_KEY] === true
        ? null
        : other.map((f) => (
            <OtherFileRow
              key={f.name}
              name={f.name}
              changed={f.changed === true}
              selected={fileSelection(f.name) === selection}
              onChoose={onChoose}
            />
          ))}
    </>
  )
}
