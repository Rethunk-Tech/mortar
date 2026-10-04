import { useLingui } from '@lingui/react/macro'
import type { ReactNode } from 'react'
import { DisabledReason } from '../shell/DisabledReason.tsx'

/** Explains a disabled control that the running game holds. */
export function LockedReason({ locked, children }: { locked: boolean; children: ReactNode }) {
  const { t } = useLingui()
  return (
    <DisabledReason title={t`Stop the game to change mods.`} disabled={locked}>
      {children}
    </DisabledReason>
  )
}
