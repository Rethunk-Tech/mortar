import { useEffect } from 'react'
import { create } from 'zustand'

interface NowState {
  now: number
}

const second = 1000
const minute = 60_000

// One shared timer per period, running only while some component watches it.
function tickingClock(period: number) {
  const useNowStore = create<NowState>(() => ({ now: Date.now() }))
  let subscribers = 0
  let timer: ReturnType<typeof setInterval> | undefined

  function start() {
    subscribers += 1
    if (subscribers === 1) {
      // The clock stood still while nobody watched it.
      useNowStore.setState({ now: Date.now() })
      timer = setInterval(() => useNowStore.setState({ now: Date.now() }), period)
    }
  }

  function stop() {
    subscribers -= 1
    if (subscribers === 0 && timer !== undefined) {
      clearInterval(timer)
      timer = undefined
    }
  }

  return { useNowStore, start, stop }
}

const minuteClock = tickingClock(minute)
const secondClock = tickingClock(second)

function useClockValue<T extends string | number | boolean>(
  clock: ReturnType<typeof tickingClock>,
  compute: (now: number) => T,
  on: boolean,
): T {
  const value = clock.useNowStore((state) => compute(state.now))
  useEffect(() => {
    if (!on) {
      return
    }
    clock.start()
    return clock.stop
  }, [clock, on])
  return value
}

/** compute's answer from the minute clock; the component re-renders only when the answer changes, so compute returns a
 * primitive (a label rather than the time). */
export function useNowValue<T extends string | number | boolean>(compute: (now: number) => T): T {
  return useClockValue(minuteClock, compute, true)
}

/** useNowValue on a clock that ticks every second while `on`; compute should return a constant while off. */
export function useSecondValue<T extends string | number | boolean>(
  compute: (now: number) => T,
  on: boolean,
): T {
  return useClockValue(secondClock, compute, on)
}
