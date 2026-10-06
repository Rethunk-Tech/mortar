import { type RefObject, useLayoutEffect, useRef, useState } from 'react'

// useWidth is the client width of the element the ref is put on, kept current as it resizes; 0 until measured.
function useWidth<T extends HTMLElement>(): [RefObject<T | null>, number] {
  const ref = useRef<T | null>(null)
  const [width, setWidth] = useState(0)
  // Measured before paint, so the first frame is laid out for the real width rather than 0.
  useLayoutEffect(() => {
    const el = ref.current
    if (!el) {
      return
    }
    const sync = () => setWidth(el.clientWidth)
    const ro = new ResizeObserver(sync)
    ro.observe(el)
    sync()
    return () => ro.disconnect()
  }, [])
  return [ref, width]
}

export { useWidth }
