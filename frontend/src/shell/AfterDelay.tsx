import { type ReactNode, useEffect, useState } from 'react'

// Long enough that a tab whose data is already at hand never flashes its placeholder.
const DELAY_MS = 200

/** Renders its children only once they have been wanted for a moment: loading placeholders go behind it. */
export function AfterDelay({ children }: { children: ReactNode }) {
  const [shown, setShown] = useState(false)
  useEffect(() => {
    const id = setTimeout(() => setShown(true), DELAY_MS)
    return () => clearTimeout(id)
  }, [])
  return shown ? children : null
}
