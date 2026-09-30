type Listener = () => void

const listeners = new Set<Listener>()

export function requestFilterFocus() {
  for (const fn of listeners) {
    fn()
  }
}

export function onFilterFocus(fn: Listener): () => void {
  listeners.add(fn)
  return () => {
    listeners.delete(fn)
  }
}
