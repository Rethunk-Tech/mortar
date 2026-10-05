import { useLingui } from '@lingui/react/macro'
import type { StartupMod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { formatDuration, whyOf } from './startupView.ts'

export function useDuration() {
  const { i18n } = useLingui()
  return (ms: number) => formatDuration(ms, i18n.locale)
}

export function useWhy() {
  const { t } = useLingui()
  const duration = useDuration()
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
        return t`slow Entry ${ms}`
      default:
        return t`time in its patches ${ms}`
    }
  }
}
