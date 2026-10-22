import type { ReactNode } from 'react'
import { useLocked } from '../useLocked.ts'
import { FixAction } from './fixActions.tsx'

/** A warning fix; a lesser one when a stronger fix sits beside it. */
export type WarningButton = (label: string, onClick: () => void, secondary?: boolean) => ReactNode

export function useWarningButton(): WarningButton {
  const locked = useLocked()
  return (label, onClick, secondary = false) => (
    <FixAction
      label={label}
      onClick={onClick}
      weight={secondary ? 'other' : 'main'}
      locked={locked}
    />
  )
}
