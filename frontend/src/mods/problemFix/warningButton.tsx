import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import type { ReactNode } from 'react'
import { DisabledReason } from '../../shell/DisabledReason.tsx'
import { useLocked } from '../useLocked.ts'

/** A warning action; outlined when a stronger fix sits beside it. */
export type WarningButton = (label: string, onClick: () => void, secondary?: boolean) => ReactNode

export function useWarningButton(): WarningButton {
  const { t } = useLingui()
  const locked = useLocked()
  return (label, onClick, secondary = false) => (
    <DisabledReason title={t`Stop the game to change mods.`} disabled={locked}>
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
    </DisabledReason>
  )
}
