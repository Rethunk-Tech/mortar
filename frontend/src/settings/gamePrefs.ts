import type { Settings } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'

export const STARDEW = 'stardew'

export interface GamePrefBlock {
  backupBeforePlay: string
  launchBackupsKept: number
  updateModsBeforePlayDefault: boolean
  runsKept: number
  consoleLogCap: number
  nxmDefaultProfile: string
  cosmeticConflicts: string
  enableRequirements: string
  missingRequirements: string
  smapiBuilds: string
  defaultLaunchMethod: string
  showSmapiConsole: boolean
  consoleLevel: string
  consoleTimestamps: boolean
  consoleFollow: boolean
}

export const defaultGamePrefs: GamePrefBlock = {
  backupBeforePlay: 'changed',
  launchBackupsKept: 5,
  updateModsBeforePlayDefault: false,
  runsKept: 20,
  consoleLogCap: 20_000,
  nxmDefaultProfile: '',
  cosmeticConflicts: 'collapsed',
  enableRequirements: 'always',
  missingRequirements: 'ask',
  smapiBuilds: 'show',
  defaultLaunchMethod: 'steam',
  showSmapiConsole: true,
  consoleLevel: 'info',
  consoleTimestamps: true,
  consoleFollow: true,
}

function on(v: boolean | null | undefined, fallback: boolean): boolean {
  if (v === null || v === undefined) {
    return fallback
  }
  return v
}

export function gamePrefs(s: Settings): GamePrefBlock {
  const got = s.games?.[STARDEW]
  if (!got) {
    return defaultGamePrefs
  }
  return {
    backupBeforePlay: got.backupBeforePlay || defaultGamePrefs.backupBeforePlay,
    launchBackupsKept: got.launchBackupsKept || defaultGamePrefs.launchBackupsKept,
    updateModsBeforePlayDefault: Boolean(got.updateModsBeforePlayDefault),
    runsKept: got.runsKept || defaultGamePrefs.runsKept,
    consoleLogCap: got.consoleLogCap || defaultGamePrefs.consoleLogCap,
    nxmDefaultProfile: got.nxmDefaultProfile ?? '',
    cosmeticConflicts: got.cosmeticConflicts || defaultGamePrefs.cosmeticConflicts,
    enableRequirements: got.enableRequirements || defaultGamePrefs.enableRequirements,
    missingRequirements: got.missingRequirements || defaultGamePrefs.missingRequirements,
    smapiBuilds: got.smapiBuilds || defaultGamePrefs.smapiBuilds,
    defaultLaunchMethod: got.defaultLaunchMethod || defaultGamePrefs.defaultLaunchMethod,
    showSmapiConsole: on(got.showSmapiConsole, true),
    consoleLevel: got.consoleLevel || defaultGamePrefs.consoleLevel,
    consoleTimestamps: on(got.consoleTimestamps, true),
    consoleFollow: on(got.consoleFollow, true),
  }
}
