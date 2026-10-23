import { type ReactNode, useRef } from 'react'

/** The menu's items as they were while it was last open. A closing menu animates out; handing it the same elements
 * lets React skip every item instead of redrawing a menu that is going away. */
export function useKeptWhileClosed(open: boolean, children: ReactNode): ReactNode {
  const kept = useRef(children)
  if (open) {
    kept.current = children
  }
  return kept.current
}
