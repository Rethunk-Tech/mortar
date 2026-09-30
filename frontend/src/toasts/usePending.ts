import { useRef, useState } from 'react'
import { reportUnexpected } from './report.ts'

function beginWork(lock: { current: boolean }) {
  if (lock.current) {
    return false
  }
  lock.current = true
  return true
}

// Runs one backend action at a time for a control, so a double click does not send it twice.
export function usePending() {
  const [pending, setPending] = useState(false)
  const inFlight = useRef(false)
  const run = (action: () => Promise<unknown>) => {
    if (!beginWork(inFlight)) {
      return
    }
    setPending(true)
    action()
      .catch(reportUnexpected)
      .finally(() => {
        inFlight.current = false
        setPending(false)
      })
  }
  return [pending, run] as const
}

export { beginWork }
