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

export function useNow() {
  const now = useNowStore((state) => state.now)
  useEffect(() => {
    start()
    return stop
  }, [])
  return now
}
