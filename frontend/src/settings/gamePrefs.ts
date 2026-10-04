import type { Settings } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'

function on(v: boolean | null | undefined, fallback: boolean): boolean {
  if (v === null || v === undefined) {
    return fallback
  }
  return v
}

const STARDEW = 'stardew'

const defaultGamePrefs: GamePrefBlock = {
  backupBeforePlay: 'changed',
  saveBackupsKept: 5,
  saveBackupHours: 0,
  saveBackupKeep: 5,
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

export interface GamePrefBlock {
  backupBeforePlay: string
  saveBackupsKept: number
  saveBackupHours: number
  saveBackupKeep: number
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

export function gamePrefs(s: Settings): GamePrefBlock {
  const got = s.games?.[STARDEW]
  if (!got) {
    return defaultGamePrefs
  }
  return {
    backupBeforePlay: got.backupBeforePlay || defaultGamePrefs.backupBeforePlay,
    saveBackupsKept: got.saveBackupsKept || defaultGamePrefs.saveBackupsKept,
    saveBackupHours: got.saveBackupHours ?? defaultGamePrefs.saveBackupHours,
    saveBackupKeep: got.saveBackupKeep || defaultGamePrefs.saveBackupKeep,
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
