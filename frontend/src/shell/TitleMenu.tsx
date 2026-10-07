import { Box, Divider, Menu, MenuItem } from '@mui/material'
import { Check } from 'lucide-react'
import type { MouseEvent, ReactNode } from 'react'

const ITEM_HEIGHT_PX = 32
const ITEM_INSET_PX = 14
const HEADING_FONT_PX = 11
const HINT_FONT_PX = 12
const HINT_GAP_PX = 24

// A dropdown under a title bar button: the menu surface, a fixed width, compact rows and small-caps headings.
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
  return (
    <Menu
      anchorEl={anchorEl}
      open={anchorEl !== null}
      onClose={onClose}
      disableEnforceFocus={true}
      anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
      transformOrigin={{ vertical: 'top', horizontal: 'left' }}
      slotProps={{
        // The backdrop starts under the title bar so its triggers still take clicks and hover while a menu is open.
        // The popover's fixed root spans the window and would take the clicks meant for them, so only its backdrop and
        // paper do.
        root: { sx: { pointerEvents: 'none' } },
        backdrop: { invisible: true, sx: { top: 'var(--title-bar)', pointerEvents: 'auto' } },
        paper: { sx: { width, maxWidth: 'calc(100vw - 16px)', mt: '4px', pointerEvents: 'auto' } },
        list: { 'aria-label': label, sx: { py: '6px' } },
      }}
    >
      {children}
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

// A menu row. `checked` makes it a radio row: the choice among a list, marked with a tick at the end.
export function TitleMenuItem({
  label,
  hint,
  checked,
  disabled,
  dim,
  accent,
  onClick,
  onContextMenu,
}: {
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
      {...(onContextMenu ? { onContextMenu } : {})}
      sx={{
        minHeight: ITEM_HEIGHT_PX,
        height: ITEM_HEIGHT_PX,
        px: `${ITEM_INSET_PX}px`,
        fontSize: 14,
        whiteSpace: 'nowrap',
        color,
      }}
    >
      <Box component="span" sx={{ minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis' }}>
        {label}
      </Box>
      {hint ? (
        <Box
          component="span"
          sx={{
            ml: 'auto',
            pl: `${HINT_GAP_PX}px`,
            fontSize: HINT_FONT_PX,
            color: 'var(--mortar-ink-sec)',
          }}
        >
          {hint}
        </Box>
      ) : null}
      {checked ? (
        <Check size={16} aria-hidden={true} style={{ marginLeft: 'auto', flexShrink: 0 }} />
      ) : null}
    </MenuItem>
  )
}
