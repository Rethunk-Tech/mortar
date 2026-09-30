import { useState } from 'react'
import { reportUnexpected } from './report.ts'

// Runs one backend action at a time for a control, so a double click does not send it twice.
export function usePending() {
  const [pending, setPending] = useState(false)
  const run = (action: () => Promise<unknown>) => {
    if (pending) {
      return
    }
    setPending(true)
    action()
      .catch(reportUnexpected)
      .finally(() => setPending(false))
  }
  return [pending, run] as const
}
