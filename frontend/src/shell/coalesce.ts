interface Slot {
  settled: Promise<void>
  // The follow-up run asked for while this one runs, shared by every later asker.
  next: Promise<void> | null
}

const ignore = () => undefined

// Runs one slow task per key at a time: an ask made while one runs waits for it and then runs once more, however
// many ask meanwhile, so a result is never older than its ask but identical slow calls do not pile up.
export function coalescer() {
  const running = new Map<string, Slot>()
  const start = (key: string, run: () => Promise<void>): Promise<void> => {
    const p = run()
    const slot: Slot = { settled: Promise.resolve(), next: null }
    slot.settled = p.then(ignore, ignore).then(() => {
      if (running.get(key) === slot) {
        running.delete(key)
      }
    })
    running.set(key, slot)
    return p
  }
  return (key: string, run: () => Promise<void>): Promise<void> => {
    const slot = running.get(key)
    if (!slot) {
      return start(key, run)
    }
    slot.next ??= slot.settled.then(() => start(key, run))
    return slot.next
  }
}
