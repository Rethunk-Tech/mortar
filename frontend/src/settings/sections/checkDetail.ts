import { useLingui } from '@lingui/react/macro'
import type { Check } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/doctor/models.ts'
import { FLATPAK, PROTON, SMAPI, STEAM } from '../../brand/productNames.ts'

function useInstalledLine() {
  const { t } = useLingui()
  return (p: Record<string, string | undefined>): string => {
    const game = p.gameVersion ? `${p.game ?? ''} ${p.gameVersion}` : (p.game ?? '')
    const path = p.path ?? ''
    const version = p.apiVersion ?? ''
    const where = p.where ?? ''
    if (version === '') {
      return where === '' ? t`${{ game }} in ${path}` : t`${{ game }} in ${path} (${where})`
    }
    return where === ''
      ? t`${{ game }} with ${{ loader: SMAPI }} ${version} in ${path}`
      : t`${{ game }} with ${{ loader: SMAPI }} ${version} in ${path} (${where})`
  }
}

// A doctor check's line in the user's language: Go sends a code and its values, and a check without a known code
// (one an older build recorded) keeps the English line Go wrote.
export function useCheckDetail() {
  const { t } = useLingui()
  const installed = useInstalledLine()
  return (c: Check): string => {
    const p = c.params ?? {}
    const game = p.game ?? ''
    const path = p.path ?? ''
    const app = p.app ?? ''
    const name = p.name ?? ''
    switch (c.code) {
      case 'mortar':
        return t`Mortar ${{ version: p.version ?? '' }} (this command ${{ otherVersion: p.commandVersion ?? '' }})`
      case 'dataDir':
        return t`Data folder: ${path}`
      case 'gameNotInstalled':
        return t`${game}: not installed`
      case 'gameInstalled':
        return installed(p)
      case 'nxmOff':
        return t`nxm:// links: off`
      case 'nxmMortar':
        return t`nxm:// links: Mortar`
      case 'nxmMortarPrevious':
        return t`nxm:// links: Mortar (other games go to ${app})`
      case 'settingsUnreadable':
        return t`settings.json is unreadable or missing`
      case 'settingsCorrupt':
        return t`settings.json is corrupt`
      case 'settingsReadable':
        return t`settings.json is readable`
      case 'settingsCopies': {
        const count = Number(p.count)
        return count === 1
          ? t`${{ count }} settings temporary or copy file is present`
          : t`${{ count }} settings temporary or copy files are present`
      }
      case 'settingsNoCopies':
        return t`No settings temporary or copy files`
      case 'profileDamaged':
        return t`Damaged profile.json: ${path}`
      case 'profilesReadable':
        return t`Profiles are readable`
      case 'controlStale':
        return t`control.json is stale`
      case 'controlCurrent':
        return t`control.json is current`
      case 'controlAbsent':
        return t`control.json is absent`
      case 'diskFull':
        return t`Data folder has no free space`
      case 'diskFree':
        return t`Data folder has free space`
      case 'storeBad':
        return t`Store index.json is unreadable or corrupt`
      case 'storeReadable':
        return t`Store index.json is readable`
      case 'nativeHostOK':
        return t`Browser extension can reach Mortar in ${app}`
      case 'nativeHostMissing':
        return t`Browser extension cannot reach Mortar in ${app} (manifest missing)`
      case 'nativeHostStale':
        return t`Browser extension cannot reach Mortar in ${app} (manifest points elsewhere)`
      case 'nativeHostUnreadable':
        return t`Browser extension cannot reach Mortar in ${app} (manifest unreadable)`
      case 'steamDeck':
        return t`${{ app: `${STEAM} Deck or ${STEAM}OS` }}: Desktop Mode launches games from Mortar`
      case 'steamDeckGameMode':
        return t`${{ app: `${STEAM} Deck` }} Game Mode: ${{ otherApp: PROTON }} games start through ${{ launcher: STEAM }} here, not from Mortar's Play`
      case 'gamescope':
        return t`Gamescope session: ${{ launcher: STEAM }} starts ${{ app: PROTON }} games here, so a direct launch from Mortar may not show a window`
      case 'flatpakLibrary':
        return t`${{ launcher: STEAM }} library ${path} is not reachable: unmounted, or outside the folders Mortar's ${{ app: FLATPAK }} may read`
      case 'protonOverride':
        return t`${game}: ${name} is overridden to native in the ${{ app: PROTON }} prefix`
      case 'protonNoPrefix':
        return t`${game}: ${{ app: PROTON }} has not created its prefix yet, so a ${{ launcher: STEAM }} launch cannot load the loader`
      case 'protonNotNative':
        return t`${game}: the ${{ app: PROTON }} prefix does not load ${name} natively, so its loader will not start`
      case 'flatpakReach':
        return t`${game}: ${{ app: FLATPAK }} ${{ launcher: STEAM }} can reach the ${{ otherApp: PROTON }} prefix`
      case 'flatpakNoLibrary':
        return t`${game}: ${{ app: FLATPAK }} ${{ launcher: STEAM }} cannot see the library that holds its ${{ otherApp: PROTON }} prefix`
      default:
        return c.detail
    }
  }
}
