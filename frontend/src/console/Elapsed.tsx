import { useEffect, useState } from 'react'

const SECOND = 1000
const PER_MINUTE = 60

/** The time since `since` as m:ss, ticking once a second; only this text redraws. */
export function Elapsed({ since }: { since: number }) {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), SECOND)
    return () => clearInterval(timer)
  }, [])
  const s = Math.max(0, Math.floor((now - since) / SECOND))
  return <>{`${Math.floor(s / PER_MINUTE)}:${String(s % PER_MINUTE).padStart(2, '0')}`}</>
}
