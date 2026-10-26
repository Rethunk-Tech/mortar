import { useEffect } from 'react'
import { create } from 'zustand'

interface NowState {
  now: number
}

const minute = 60_000
const useNowStore = create<NowState>(() => ({ now: Date.now() }))
let subscribers = 0
let timer: ReturnType<typeof setInterval> | undefined

function start() {
  subscribers += 1
  if (subscribers === 1) {
    // The clock stood still while nobody watched it.
    useNowStore.setState({ now: Date.now() })
    timer = setInterval(() => useNowStore.setState({ now: Date.now() }), minute)
  }
}

function stop() {
  subscribers -= 1
  if (subscribers === 0 && timer !== undefined) {
    clearInterval(timer)
    timer = undefined
  }
}

/** compute's answer from the minute clock; the component re-renders only when the answer changes, so compute returns a
 * primitive (a label rather than the time). */
export function useNowValue<T extends string | number | boolean>(compute: (now: number) => T): T {
  const value = useNowStore((state) => compute(state.now))
  useEffect(() => {
    start()
    return stop
  }, [])
  return value
}
