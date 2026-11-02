import { type RefObject, useLayoutEffect, useState } from 'react'

/** The client size of the element the ref points at, kept current as it resizes; 0 until measured. */
export function useElementSize(ref: RefObject<HTMLElement | null>) {
  const [size, setSize] = useState({ width: 0, height: 0 })
  // Measured before paint, so the first frame is laid out for the real size rather than 0.
  useLayoutEffect(() => {
    const el = ref.current
    if (!el) {
      return
    }
    const measure = () =>
      setSize((prev) =>
        prev.width === el.clientWidth && prev.height === el.clientHeight
          ? prev
          : { width: el.clientWidth, height: el.clientHeight },
      )
    const ro = new ResizeObserver(measure)
    ro.observe(el)
    measure()
    return () => ro.disconnect()
  }, [ref])
  return size
}
