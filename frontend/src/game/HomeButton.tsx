import { Button, type ButtonProps, Menu } from '@mui/material'
import { ChevronDown } from 'lucide-react'
import type { ReactNode } from 'react'
import { Tooltip } from '../shell/Tooltip.tsx'
import { usePopoverAnchor } from '@rethunk/react-load-hooks'
import { space } from '../theme/density.ts'
import { iconWhenCompact } from './compact.ts'

// A Home action: outlined, a leading icon, 36px tall. Compact, it folds into an icon button that keeps its label.
export function HomeButton({
  icon,
  children,
  ...rest
}: Omit<ButtonProps, 'startIcon' | 'variant' | 'color' | 'children'> & {
  icon: ReactNode
  children: string
}) {
  return (
    <Tooltip title={children}>
      <Button
        variant="outlined"
        color="inherit"
        startIcon={icon}
        aria-label={children}
        sx={{
          height: space.control,
          borderColor: 'var(--mortar-hairline-22)',
          whiteSpace: 'nowrap',
          ...iconWhenCompact,
        }}
        {...rest}
      >
        <span className="label">{children}</span>
      </Button>
    </Tooltip>
  )
}

/** A HomeButton that opens a menu; `menu` is built only while it is open and gets the function that closes it. */
export function HomeMenuButton({
  icon,
  label,
  menu,
  onOpen,
}: {
  icon: ReactNode
  label: string
  menu: (close: () => void) => ReactNode
  onOpen?: () => void
}) {
  const { anchor, show, hide } = usePopoverAnchor()
  const close = () => hide()
  return (
    <>
      <HomeButton
        icon={icon}
        endIcon={<ChevronDown size={14} aria-hidden={true} />}
        aria-haspopup="menu"
        aria-expanded={anchor !== null}
        onClick={(e) => {
          show(e)
          onOpen?.()
        }}
      >
        {label}
      </HomeButton>
      <Menu open={anchor !== null} anchorEl={anchor} onClose={close}>
        {anchor === null ? null : menu(close)}
      </Menu>
    </>
  )
}
