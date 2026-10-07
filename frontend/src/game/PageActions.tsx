import type { ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { usePageActionsSlot } from './pageActions.ts'

// A tab's header buttons, rendered into the shared TabHeader's right-hand slot.
export function PageActions({ children }: { children: ReactNode }) {
  const slot = usePageActionsSlot((s) => s.slot)
  return slot ? createPortal(children, slot) : null
}
