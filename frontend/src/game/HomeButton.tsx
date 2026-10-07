import { Button, type ButtonProps } from '@mui/material'
import type { ReactNode } from 'react'
import { space } from '../theme/density.ts'

// A Home action: outlined, a leading icon, 36px tall.
export function HomeButton({
  icon,
  children,
  ...rest
}: Omit<ButtonProps, 'startIcon' | 'variant' | 'color'> & { icon: ReactNode }) {
  return (
    <Button
      variant="outlined"
      color="inherit"
      startIcon={icon}
      sx={{ height: space.control, borderColor: 'var(--mortar-hairline-22)', whiteSpace: 'nowrap' }}
      {...rest}
    >
      {children}
    </Button>
  )
}
