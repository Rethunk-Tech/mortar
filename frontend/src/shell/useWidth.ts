import { type RefObject, useEffect, useRef, useState } from 'react'

// useWidth is the client width of the element the ref is put on, kept current as it resizes; 0 until measured.
function useWidth<T extends HTMLElement>(): [RefObject<T | null>, number] {
  const ref = useRef<T | null>(null)
  const [width, setWidth] = useState(0)
  useEffect(() => {
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
