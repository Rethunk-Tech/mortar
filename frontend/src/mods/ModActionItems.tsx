import { useLingui } from '@lingui/react/macro'
import { Box, Collapse, ListItemIcon, ListItemText, MenuItem } from '@mui/material'
import { ChevronDown, ChevronRight, CopyPlus, PackagePlus, Tag, Trash2, Users } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import { MenuAction } from '../shell/MenuAction.tsx'
import { MenuRule } from '../shell/TitleMenu.tsx'
import { space } from '../theme/density.ts'
import { ICON_SIZE } from './menu.ts'
import type { ModAction } from './modActions.ts'

// One row that unfolds the two actions on this mod's copies in the game's other profiles; the menu stays open.
function OtherProfilesGroup({
  locked,
  tooltip,
  closeThen,
  onAlsoAdd,
  onRemoveOther,
  labels,
}: {
  locked: boolean
  tooltip: string
  closeThen: (fn: () => void) => () => void
  onAlsoAdd: () => void
  onRemoveOther: () => void
  labels: { alsoAdd: string }
}) {
  const { t } = useLingui()
  const [open, setOpen] = useState(false)
  return (
    <>
      <MenuItem aria-expanded={open} onClick={() => setOpen(!open)}>
        <ListItemIcon sx={{ color: 'inherit' }}>
          <Users size={ICON_SIZE} />
        </ListItemIcon>
        <ListItemText>{t`Other profiles`}</ListItemText>
        {open ? <ChevronDown size={ICON_SIZE} /> : <ChevronRight size={ICON_SIZE} />}
      </MenuItem>
      <Collapse in={open} unmountOnExit={true}>
        <Box sx={{ pl: space.pad }}>
          <MenuAction
            disabled={locked}
            tooltip={tooltip}
            icon={<CopyPlus size={ICON_SIZE} />}
            label={labels.alsoAdd}
            onClick={closeThen(onAlsoAdd)}
          />
          <MenuAction
            disabled={locked}
            tooltip={tooltip}
            icon={<Trash2 size={ICON_SIZE} />}
            label={t`Remove from other profiles…`}
            onClick={closeThen(onRemoveOther)}
          />
        </Box>
      </Collapse>
    </>
  )
}

function RemoveMenuItem({
  item,
  locked,
  tooltip,
  close,
}: {
  item: { label: string; icon: ReactNode; run: () => void }
  locked: boolean
  tooltip: string
  close: () => void
}) {
  return (
    <MenuAction
      disabled={locked}
      tone="error"
      icon={item.icon}
      label={item.label}
      tooltip={locked ? tooltip : undefined}
      onClick={() => {
        close()
        item.run()
      }}
    />
  )
}

function ModActionItems({
  actions,
  items,
  close,
  locked,
  hasFomod,
  onSetCategory,
  onAlsoAdd,
  onAddBundle,
  onRemoveOther,
  labels,
  splitCombine,
  trailing,
}: {
  actions: ModAction[]
  items: Record<ModAction | 'reinstall', { label: string; icon: ReactNode; run: () => void }>
  close: () => void
  locked: boolean
  hasFomod: boolean
  onSetCategory: () => void
  onAlsoAdd: () => void
  onAddBundle: () => void
  onRemoveOther: () => void
  labels: { category: string; alsoAdd: string; addBundle: string }
  splitCombine: ReactNode
  trailing: ReactNode[]
}) {
  const { t } = useLingui()
  const lockedTip = t`Stop the game to change mods.`
  const closeThen = (fn: () => void) => () => {
    close()
    fn()
  }
  const renderAction = (action: ModAction | 'reinstall', disabled = false) => (
    <MenuAction
      key={action}
      disabled={disabled}
      tooltip={lockedTip}
      icon={items[action].icon}
      label={items[action].label}
      onClick={closeThen(items[action].run)}
    />
  )
  const has = (action: ModAction) => actions.includes(action)
  const result: ReactNode[] = []
  if (has('toggle')) {
    result.push(renderAction('toggle', locked))
  }
  if (has('details')) {
    result.push(renderAction('details'))
  }
  result.push(<MenuRule key="source-divider" />)
  if (has('page')) {
    result.push(renderAction('page'))
  }
  if (has('files')) {
    result.push(renderAction('files'))
    if (hasFomod) {
      result.push(renderAction('reinstall', locked))
    }
    result.push(<MenuRule key="organisation-divider" />)
    result.push(
      <MenuAction
        key="category"
        icon={<Tag size={ICON_SIZE} />}
        label={labels.category}
        onClick={closeThen(onSetCategory)}
      />,
    )
    result.push(
      <MenuAction
        key="add-bundle"
        disabled={locked}
        tooltip={lockedTip}
        icon={<PackagePlus size={ICON_SIZE} />}
        label={labels.addBundle}
        onClick={closeThen(onAddBundle)}
      />,
    )
    if (has('pin')) {
      result.push(renderAction('pin'))
    }
    if (has('skip')) {
      result.push(renderAction('skip'))
    }
  }
  result.push(<MenuRule key="other-profiles-divider" />)
  result.push(
    <OtherProfilesGroup
      key="other-profiles"
      locked={locked}
      tooltip={lockedTip}
      closeThen={closeThen}
      onAlsoAdd={onAlsoAdd}
      onRemoveOther={onRemoveOther}
      labels={labels}
    />,
  )
  if (splitCombine) {
    result.push(splitCombine)
  }
  result.push(...trailing)
  if (has('remove')) {
    result.push(<MenuRule key="remove-divider" />)
    result.push(
      <RemoveMenuItem
        key="remove"
        item={items.remove}
        locked={locked}
        tooltip={lockedTip}
        close={close}
      />,
    )
  }
  return result
}

export { ModActionItems }
