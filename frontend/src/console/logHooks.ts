import { useLingui } from '@lingui/react/macro'
import { useMemo } from 'react'
import { Level } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
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

export function useLevelNames(): Record<Level, string> {
  const { t } = useLingui()
  return {
    [Level.$zero]: '',
    [Level.Trace]: t`Trace`,
    [Level.Debug]: t`Debug`,
    [Level.Info]: t`Info`,
    [Level.Warn]: t`Warn`,
    [Level.Error]: t`Error`,
    [Level.Alert]: t`Alert`,
  }
}
