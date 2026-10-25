import { Box, Divider, Menu, MenuItem } from '@mui/material'
import { Check } from 'lucide-react'
import type { MouseEvent, ReactNode } from 'react'
import { space } from '../theme/density.ts'
import { useKeptWhileClosed } from './keptWhileClosed.ts'
import { pickWithoutFocus } from './pickWithoutFocus.ts'

const ITEM_INSET_PX = 14
const HEADING_FONT_PX = 11
const HINT_FONT_PX = 12
const HINT_GAP_PX = 24
const ICON_COLUMN_PX = 22

// A dropdown under a title bar button: the menu surface, a minimum width, compact rows and small-caps headings.
export function TitleMenu({
  anchorEl,
  onClose,
  label,
  width,
  children,
}: {
  anchorEl: HTMLElement | null
  onClose: () => void
  label: string
  width: number
  children: ReactNode
}) {
  const items = useKeptWhileClosed(anchorEl !== null, children)
  return (
    <Menu
      anchorEl={anchorEl}
      open={anchorEl !== null}
      onClose={onClose}
      // Kept mounted: each opening then redraws only the items that changed, not every item twice.
      keepMounted={true}
      disableEnforceFocus={true}
      anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
      transformOrigin={{ vertical: 'top', horizontal: 'left' }}
      slotProps={{
        // The backdrop starts under the title bar so its triggers still take clicks and hover while a menu is open.
        // The popover's fixed root spans the window and would take the clicks meant for them, so only its backdrop and
        // paper do.
        root: { sx: { pointerEvents: 'none' } },
        backdrop: { invisible: true, sx: { top: 'var(--title-bar)', pointerEvents: 'auto' } },
        // width is a floor: a menu grows to its longest label, up to a cap that still truncates a long profile name.
        paper: {
          sx: {
            minWidth: width,
            width: 'max-content',
            maxWidth: 'min(440px, calc(100vw - 16px))',
            mt: '4px',
            pointerEvents: 'auto',
          },
        },
        list: { 'aria-label': label, sx: { py: space.menuY } },
      }}
    >
      {items}
    </Menu>
  )
}

export function MenuHeading({ children }: { children: ReactNode }) {
  return (
    <Box
      role="presentation"
      sx={{
        px: `${ITEM_INSET_PX}px`,
        pt: '6px',
        pb: '2px',
        fontSize: HEADING_FONT_PX,
        fontWeight: 700,
        letterSpacing: '0.08em',
        textTransform: 'uppercase',
        color: 'var(--mortar-ink-sec)',
      }}
    >
      {children}
    </Box>
  )
}

export function MenuRule() {
  return <Divider sx={{ my: '4px' }} />
}

// The row's parts are spans styled from the row: a menu redraws its rows as it opens, and each Box is three components.
const itemSx = {
  minHeight: 0,
  px: `${ITEM_INSET_PX}px`,
  fontSize: 'medium.fontSize',
  whiteSpace: 'nowrap',
  '& .title-menu-icon': {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    width: ICON_COLUMN_PX,
    mr: '10px',
    flexShrink: 0,
  },
  '& .title-menu-label': { minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis' },
  '& .title-menu-hint': {
    ml: 'auto',
    pl: `${HINT_GAP_PX}px`,
    fontSize: HINT_FONT_PX,
    color: 'var(--mortar-ink-sec)',
  },
} as const

// Only a label cut off at the menu's width cap needs its full text on hover.
const showCutLabel = (e: MouseEvent<HTMLElement>) => {
  const el = e.currentTarget.querySelector<HTMLElement>('.title-menu-label')
  if (el) {
    el.title = el.scrollWidth > el.clientWidth ? (el.textContent ?? '') : ''
  }
}

// A menu row. `checked` makes it a radio row: the choice among a list, marked with a tick at the end.
export function TitleMenuItem({
  icon,
  label,
  hint,
  checked,
  disabled,
  dim,
  accent,
  onClick,
  onContextMenu,
}: {
  // Leads the row in a fixed column, so every row of a menu lines up.
  icon?: ReactNode
  label: ReactNode
  hint?: ReactNode
  checked?: boolean
  disabled?: boolean
  dim?: boolean
  accent?: boolean
  onClick: () => void
  onContextMenu?: (e: MouseEvent<HTMLElement>) => void
}) {
  let color = 'inherit'
  if (accent) {
    color = 'var(--mortar-accent-ink)'
  } else if (dim) {
    color = 'var(--mortar-ink-sec)'
  }
  return (
    <MenuItem
      {...(checked === undefined ? {} : { role: 'menuitemradio', 'aria-checked': checked })}
      disabled={disabled}
      onClick={onClick}
      onMouseDown={pickWithoutFocus}
      onMouseEnter={showCutLabel}
      {...(onContextMenu ? { onContextMenu } : {})}
      sx={[itemSx, { color }]}
    >
      {icon ? (
        <span className="title-menu-icon" aria-hidden={true}>
          {icon}
        </span>
      ) : null}
      <span className="title-menu-label">{label}</span>
      {hint ? <span className="title-menu-hint">{hint}</span> : null}
      {checked ? (
        <Check size={16} aria-hidden={true} style={{ marginLeft: 'auto', flexShrink: 0 }} />
      ) : null}
    </MenuItem>
  )
}
