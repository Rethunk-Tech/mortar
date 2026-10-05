import { msg } from '@lingui/core/macro'
import { useRef, useState } from 'react'
import { i18n } from '../i18n/index.ts'
import { reportError, toastError } from './report.ts'

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
  const run = (
    action: () => Promise<unknown>,
    options?: { errorTitle?: string | undefined },
  ): void => {
    if (!beginWork(inFlight)) {
      return
    }
    setPending(true)
    const title = options?.errorTitle
    const retry = () => run(action, options)
    action()
      .catch(
        title === undefined
          ? (e: unknown) => toastError(i18n._(msg`Something went wrong`), e, { retry })
          : reportError(title, retry),
      )
      .finally(() => {
        inFlight.current = false
        setPending(false)
      })
  }
  return [pending, run] as const
}

export { beginWork }
