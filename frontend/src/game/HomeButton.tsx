import { Button, type ButtonProps, Tooltip } from '@mui/material'
import type { ReactNode } from 'react'
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
