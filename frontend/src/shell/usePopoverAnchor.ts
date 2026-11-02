import { type MouseEvent, useState } from 'react'

/** The element a popover or menu hangs from: set by the click that opens it, null while closed. */
export function usePopoverAnchor() {
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  return {
    anchor,
    show: (e: MouseEvent<HTMLElement>) => setAnchor(e.currentTarget),
    hide: () => setAnchor(null),
  }
}
