import { useLingui } from '@lingui/react/macro'
import { MONO_RUNTIME, SMAPI, UNITY, WINDOWS } from '../brand/productNames.ts'

type Params = { [_ in string]?: string } | null | undefined

export type ReasonText = (code: string | undefined, params: Params, fallback: string) => string

// A reason Mortar wrote in Go (a load failure, a crash cause or hint), worded in the user's language from its code and
// values. A reason without a known code (quoted from a log, or saved by an older build) shows as written.
export function useReasonText(): ReasonText {
  const { t } = useLingui()
  return (code, params, fallback) => {
    const p = params ?? {}
    const list = p.list ?? ''
    const name = p.name ?? ''
    const module = p.module ?? ''
    switch (code) {
      case 'missingDependencies':
        return t`missing dependencies: ${list}`
      case 'incompatibleDependencies':
        return t`incompatible dependencies: ${list}`
      case 'incompatibleWith':
        return t`incompatible with: ${list}`
      case 'sameCopyLoaded':
        return t`the same copy loaded instead from ${{ path: p.path ?? '' }}`
      case 'newerCopyLoaded':
        return t`a newer copy loaded instead: ${{ version: p.version ?? '' }}`
      case 'moduleFailed':
        return t`module ${name} failed`
      case 'moduleFailedWith':
        return t`module ${name} failed: ${{ error: p.error ?? '' }}`
      case 'patchMatchedNothing':
        return t`a patch for ${name} matched nothing in the game's code`
      case 'gameVersionOlder':
        return t`game version ${{ version: p.version ?? '' }} is older than the loader accepts (${{ otherVersion: p.accepted ?? '' }})`
      case 'off':
        return t`Antivirus scanning is off`
      case 'noCommand':
        return t`No command set`
      case 'none':
        return t`No antivirus found`
      case 'windowsAmsi':
        return t`${{ os: WINDOWS }} antivirus (AMSI)`
      case 'namedAmsi':
        return t`${{ names: p.names ?? '' }} (AMSI)`
      case 'orphan':
        return t`Brought in for other mods, and none that is enabled needs it`
      case 'recolourMissing':
        return t`Made for ${name}, which is not enabled`
      case 'tilesheetsUnused':
        return t`Tilesheets not used by any installed mod`
      case 'tilesheetsOnlyDisabled':
        return t`Used only by disabled mods: ${{ names: p.names ?? '' }}`
      case 'alreadyInProfile':
        return t`already in this profile as ${name}`
      case 'upToDate':
        return t`already up to date in this profile`
      case 'sameMod':
        return t`same mod as ${name}`
      case 'noSmapiMod':
        return t`No ${{ loader: SMAPI }} mod was found`
      case 'notRemoved':
        return t`added, but it could not be removed from the game's Mods folder: ${{ error: p.error ?? '' }}`
      case 'missing-file':
        return t`A file it needs could not be opened.`
      case 'asset-load':
        return t`It could not load an asset.`
      case 'mod-exception':
        return p.exception
          ? t`${{ exception: p.exception }} at ${{ frame: p.frame ?? '' }}`
          : t`It encountered an error.`
      case 'modCode':
        return t`The game crashed inside this mod's code.`
      case 'gameCrash':
        return t`The game itself crashed (${module}), not a mod's code.`
      case 'unityCrash':
        return t`${{ app: UNITY }} crashed (${module}), not a mod's code.`
      case 'monoCrash':
        return t`The ${{ app: MONO_RUNTIME }} runtime crashed (${module}), not a mod's code.`
      default:
        return fallback
    }
  }
}
