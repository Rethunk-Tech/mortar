import { Button } from '@mui/material'
import type { ReactNode } from 'react'
import { useLocked } from '../useLocked.ts'

export type WarningButton = (label: string, onClick: () => void) => ReactNode

export function useWarningButton(): WarningButton {
  const locked = useLocked()
  return (label, onClick) => (
    <Button
      size="small"
      variant="contained"
      color="warning"
      disabled={locked}
      onClick={onClick}
      sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
    >
      {label}
    </Button>
  )
}
