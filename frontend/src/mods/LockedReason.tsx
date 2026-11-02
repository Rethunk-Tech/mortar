import { useLingui } from '@lingui/react/macro'
import type { ReactNode } from 'react'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { lockedHint } from './lockedHint.ts'

/** Explains a disabled control that the running game holds. */
export function LockedReason({ locked, children }: { locked: boolean; children: ReactNode }) {
  const { t } = useLingui()
  return (
    <DisabledReason title={t(lockedHint)} disabled={locked}>
      {children}
    </DisabledReason>
  )
}
