import { useLingui } from '@lingui/react/macro'
import type { StartupMod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { useProfileLoader } from '../profiles/store.ts'
import { formatDuration, whyOf } from './startupView.ts'

/** Whether the open profile's startup reports come from SMAPI, whose bridge times more than BepInEx's patcher: Entry,
 * events and assets, and a sampled share on a measured launch. */
export function useSmapiStartup(): boolean {
  return useProfileLoader()?.id === 'smapi'
}

/** Whether the open profile's companion measures in game and answers as data, rather than through SMAPI's console. */
export function usePerfQuery(): boolean {
  return useProfileLoader()?.perf === true
}

export function useDuration() {
  const { i18n } = useLingui()
  return (ms: number) => formatDuration(ms, i18n.locale)
}

export function useWhy() {
  const { t } = useLingui()
  const duration = useDuration()
  const smapi = useSmapiStartup()
  return (mod: StartupMod): string => {
    const why = whyOf(mod)
    if (!why) {
      return '—'
    }
    const ms = duration(why.ms)
    switch (why.kind) {
      case 'event':
        return t`slow ${why.event ?? ''} ${ms}`
      case 'assets':
        return t`assets and packs ${ms}`
      case 'entry':
        return smapi ? t`slow Entry ${ms}` : t`slow load ${ms}`
      default:
        return t`time in its patches ${ms}`
    }
  }
}
