import { useLingui } from '@lingui/react/macro'
import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'

// A line of Mortar's own in the console, worded in the user's language from the code and values Go sent. A line
// without a known code (the game's, a mod's, or one an older build saved) shows its message as written.
export function useLineText() {
  const { t } = useLingui()
  return (e: Entry): string => {
    const p = e.params ?? {}
    const game = p.game ?? ''
    const error = p.error ?? ''
    const signal = p.signal ?? ''
    const code = p.code ?? ''
    const after = p.duration ? ` ${t`after ${{ duration: p.duration }}`}` : ''
    switch (e.code) {
      case 'settingsMissing':
        return t`startup_preferences is missing; skipped profile game settings.`
      case 'backupFailed':
        return t`Could not back up saves before playing: ${error}.`
      case 'vanilla':
        return t`Started without mods`
      case 'loaderExited':
        return t`${{ loader: p.loader ?? '' }} exited with code ${code}.`
      case 'settingsRestoreFailed':
        return t`Could not restore profile game settings: ${error}.`
      case 'gameClosed':
        return `${t`${game} closed`}${after}.`
      case 'gameStopped':
        return `${t`${game} was stopped from Mortar`}${after}.`
      case 'gameCrashed': {
        let exit = t`${game} crashed; exited with code ${code}`
        if (signal !== '') {
          exit =
            code === '0'
              ? t`${game} crashed; exited with ${signal}`
              : t`${game} crashed; exited with code ${code} (${signal})`
        }
        return `${exit}${after}.`
      }
      default:
        return e.message
    }
  }
}
