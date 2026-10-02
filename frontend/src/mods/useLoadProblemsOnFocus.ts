import { useEffect } from 'react'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useMods } from './store.ts'

const FOCUS_DEBOUNCE_MS = 400

export function useLoadProblemsOnFocus() {
  const loadProblems = useMods((s) => s.loadProblems)
  const openId = useProfiles((s) => s.openId)
  // Only the Mods tab reloads on a profile switch, so a Problems tab that stays open loads the new profile itself:
  // its fix buttons need that profile's mods as well as its problems.
  useEffect(() => {
    const { modsFor, problemsFor, load } = useMods.getState()
    if (openId && modsFor !== openId) {
      load().catch(reportUnexpected)
    } else if (openId && problemsFor !== openId) {
      loadProblems().catch(reportUnexpected)
    }
  }, [loadProblems, openId])
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
