import { useMediaQuery } from '@mui/material'
import { useSettings } from '../settings/store.ts'

// The Reduce motion setting: always, never, or the system's choice.
export function honourReducedMotion(mode: string, os: boolean) {
  if (mode === 'always') {
    return true
  }
  if (mode === 'never') {
    return false
  }
  return os
}

export function useReducedMotion(): boolean {
  const mode = useSettings((s) => s.reduceMotion)
  const os = useMediaQuery('(prefers-reduced-motion: reduce)', { noSsr: true })
  return honourReducedMotion(mode || 'system', os)
}
