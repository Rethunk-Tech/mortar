import { type RefObject, useLayoutEffect, useRef, useState } from 'react'
import { flushSync } from 'react-dom'

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
    // Rendered before the browser paints the resize: a default-priority update would paint a frame laid out for the
    // old width, so an opening details panel flashed the list's full column set and a sideways scrollbar.
    const ro = new ResizeObserver(() => flushSync(() => setWidth(el.clientWidth)))
    ro.observe(el)
    setWidth(el.clientWidth)
    return () => ro.disconnect()
  }, [])
  return [ref, width]
}

export { useWidth }
