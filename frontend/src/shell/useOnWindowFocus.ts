import { useEffect } from 'react'

// Runs fn whenever the window regains focus, so a change made elsewhere (a launcher or game installed, a file edited in
// an editor) shows at once. Pass a stable fn: a new one resubscribes.
export function useOnWindowFocus(fn: () => void, active = true) {
  useEffect(() => {
    if (!active) {
      return
    }
    globalThis.addEventListener('focus', fn)
    return () => globalThis.removeEventListener('focus', fn)
  }, [fn, active])
}
