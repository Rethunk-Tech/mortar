import { ListSubheader } from '@mui/material'
import type { ReactNode } from 'react'

export function MenuHeading({ children }: { children: ReactNode }) {
  return (
    <ListSubheader sx={{ lineHeight: '28px', textTransform: 'uppercase' }}>
      {children}
    </ListSubheader>
  )
}
