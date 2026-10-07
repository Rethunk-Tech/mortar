import { Events } from '@wailsio/runtime'
import { useEffect } from 'react'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useMods } from './store.ts'

const FOCUS_DEBOUNCE_MS = 400

// The hook is mounted by the sidebar and by both Problems views, so each would add its own focus listener and one
// focus would start one scan per listener. The listeners are shared: the first user adds them, the last removes them.
let users = 0
let stopWatching: (() => void) | undefined

function watchFocus(loadProblems: () => Promise<void>) {
  users += 1
  if (users === 1) {
    stopWatching = attachFocusListeners(loadProblems)
  }
  return () => {
    users -= 1
    if (users === 0) {
      stopWatching?.()
      stopWatching = undefined
    }
  }
}

function attachFocusListeners(loadProblems: () => Promise<void>) {
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
  // A scan that was waiting on the keyring prompt retries once the user unlocks it.
  const offUnlocked = Events.On('keyring:unlocked', scan)
  globalThis.addEventListener('focus', scan)
  document.addEventListener('visibilitychange', onVisible)
  return () => {
    globalThis.clearTimeout(timer)
    offUnlocked()
    globalThis.removeEventListener('focus', scan)
    document.removeEventListener('visibilitychange', onVisible)
  }
}

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
  useEffect(() => watchFocus(loadProblems), [loadProblems])
}
