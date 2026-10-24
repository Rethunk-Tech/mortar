import { useRef, useState } from 'react'
import { errorMessage, expectedFailure, reportError, toastError } from './report.ts'

function beginWork(lock: { current: boolean }) {
  if (lock.current) {
    return false
  }
  lock.current = true
  return true
}

// failureSink reports a failed action, or with quietExpected leaves an expected one to the offline banner or unlock retry.
const failureSink =
  (report: (e: unknown) => void, quietExpected = false) =>
  (e: unknown) => {
    if (!(quietExpected && expectedFailure(e))) {
      report(e)
    }
  }

// Runs one backend action at a time for a control, so a double click does not send it twice.
export function usePending() {
  const [pending, setPending] = useState(false)
  const inFlight = useRef(false)
  const run = (
    action: () => Promise<unknown>,
    options?: { errorTitle?: string | undefined; quietExpected?: boolean },
  ): void => {
    if (!beginWork(inFlight)) {
      return
    }
    setPending(true)
    const title = options?.errorTitle
    const retry = () => run(action, options)
    const report =
      title === undefined
        ? (e: unknown) => toastError(errorMessage(null), e, { retry })
        : reportError(title, retry)
    action()
      .catch(failureSink(report, options?.quietExpected))
      .finally(() => {
        inFlight.current = false
        setPending(false)
      })
  }
  return [pending, run] as const
}

export { beginWork, failureSink }
