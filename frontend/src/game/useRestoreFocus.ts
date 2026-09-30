import { type RefObject, useEffect, useRef } from 'react'

// When `active` turns off and the element that held focus went with it, focus returns to `target`.
export function useRestoreFocus(active: boolean, target: RefObject<HTMLElement | null>) {
  const was = useRef(active)
  useEffect(() => {
    if (was.current && !active && document.activeElement === document.body) {
      target.current?.focus()
    }
    was.current = active
  }, [active, target])
}
