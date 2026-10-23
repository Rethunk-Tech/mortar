import { useLingui } from '@lingui/react/macro'
import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'

function useCrashedLine() {
  const { t } = useLingui()
  return (p: Record<string, string | undefined>): string => {
    const game = p.game ?? ''
    const code = p.code ?? ''
    const signal = p.signal ?? ''
    const duration = p.duration ?? ''
    if (signal === '') {
      return duration === ''
        ? t`${game} crashed; exited with code ${code}.`
        : t`${game} crashed; exited with code ${code} after ${duration}.`
    }
    if (code === '0') {
      return duration === ''
        ? t`${game} crashed; exited with ${signal}.`
        : t`${game} crashed; exited with ${signal} after ${duration}.`
    }
    return duration === ''
      ? t`${game} crashed; exited with code ${code} (${signal}).`
      : t`${game} crashed; exited with code ${code} (${signal}) after ${duration}.`
  }
}

// A line of Mortar's own in the console, worded in the user's language from the code and values Go sent. A line
// without a known code (the game's, a mod's, or one an older build saved) shows its message as written.
export function useLineText() {
  const { t } = useLingui()
  const crashed = useCrashedLine()
  return (e: Entry): string => {
    const p = e.params ?? {}
    const game = p.game ?? ''
    const error = p.error ?? ''
    const code = p.code ?? ''
    const duration = p.duration ?? ''
    switch (e.code) {
      case 'settingsMissing':
        return t`startup_preferences is missing; skipped profile game settings.`
      case 'backupFailed':
        return t`Could not back up saves before playing: ${error}.`
      case 'vanilla':
        return t`Started without mods`
      case 'launchCancelled':
        return t`Launch cancelled.`
      case 'loaderExited':
        return t`${{ loader: p.loader ?? '' }} exited with code ${code}.`
      case 'settingsRestoreFailed':
        return t`Could not restore profile game settings: ${error}.`
      case 'gameClosed':
        return duration === '' ? t`${game} closed.` : t`${game} closed after ${duration}.`
      case 'gameStopped':
        return duration === ''
          ? t`${game} was stopped from Mortar.`
          : t`${game} was stopped from Mortar after ${duration}.`
      case 'gameCrashed':
        return crashed(p)
      default:
        return e.message
    }
  }
}
