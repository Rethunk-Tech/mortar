import { useEffect } from 'react'

// Re-runs detection when the window regains focus, so installing a launcher or a game elsewhere shows at once.
export function useRefreshOnFocus(refresh: () => void, active = true) {
  useEffect(() => {
    if (!active) {
      return
    }
    window.addEventListener('focus', refresh)
    return () => window.removeEventListener('focus', refresh)
  }, [refresh, active])
}
