import { useEffect } from 'react'
import { reportUnexpected } from '../toasts/report.ts'
import { useMods } from './store.ts'

const FOCUS_DEBOUNCE_MS = 400

export function useLoadProblemsOnFocus() {
  const loadProblems = useMods((s) => s.loadProblems)
  useEffect(() => {
    let timer: ReturnType<typeof globalThis.setTimeout> | undefined
    const scan = () => {
      globalThis.clearTimeout(timer)
      timer = globalThis.setTimeout(() => {
        loadProblems().catch(reportUnexpected)
      }, FOCUS_DEBOUNCE_MS)
    }
    const onVisible = () => {
      if (document.visibilityState === 'visible') {
        scan()
      }
    }
    globalThis.addEventListener('focus', scan)
    document.addEventListener('visibilitychange', onVisible)
    return () => {
      globalThis.clearTimeout(timer)
      globalThis.removeEventListener('focus', scan)
      document.removeEventListener('visibilitychange', onVisible)
    }
  }, [loadProblems])
}
