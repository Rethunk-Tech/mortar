import { Button } from '@mui/material'
import type { ReactNode } from 'react'
import { useLocked } from '../useLocked.ts'

/** A warning action; outlined when a stronger fix sits beside it. */
export type WarningButton = (label: string, onClick: () => void, secondary?: boolean) => ReactNode

export function useWarningButton(): WarningButton {
  const locked = useLocked()
  return (label, onClick, secondary = false) => (
    <Button
      size="small"
      variant={secondary ? 'outlined' : 'contained'}
      color="warning"
      disabled={locked}
      onClick={onClick}
      sx={{ flexShrink: 0 }}
    >
      {label}
    </Button>
  )
}
