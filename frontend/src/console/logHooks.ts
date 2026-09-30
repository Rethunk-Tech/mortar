import { useMemo } from 'react'
import { useProfiles } from '../profiles/store.ts'
import { shownLog, visible } from './filter.ts'
import { useConsole } from './store.ts'

function useMine() {
  return useConsole((s) => s.shown.profile) === useProfiles((s) => s.openId)
}

export function useShownEntries() {
  const entries = useConsole((s) => s.entries)
  const mine = useMine()
  return useMemo(() => shownLog(entries, mine), [mine, entries])
}

export function useVisible() {
  const entries = useShownEntries()
  const filters = useConsole((s) => s.filters)
  return useMemo(() => visible(entries, filters), [entries, filters])
}
