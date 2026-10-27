import type { Settings } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { currentGame } from '../nav/currentGame.ts'
import { cachedSpecDefault } from './prefSpecs.ts'

// Defaults live in Go's registry; the specs are loaded before the settings are, so these read them.
const text = (key: string, game: string) => cachedSpecDefault(key, game) ?? ''
const count = (key: string, game: string) => Number(cachedSpecDefault(key, game) ?? 0)
const flag = (key: string, game: string) => cachedSpecDefault(key, game) === 'true'

function on(v: boolean | null | undefined, fallback: boolean): boolean {
  if (v === null || v === undefined) {
    return fallback
  }
  return v
}

export interface GamePrefBlock {
  backupBeforePlay: string
  saveBackupsKept: number
  saveBackupHours: number
  saveBackupKeep: number
  updateModsBeforePlayDefault: boolean
  pauseDownloadsWhilePlaying: boolean
  runsKept: number
  consoleLogCap: number
  nxmDefaultProfile: string
  cosmeticConflicts: string
  enableRequirements: string
  disableDependencies: string
  missingRequirements: string
  smapiBuilds: string
  defaultLaunchMethod: string
  startFirst: string
  showSmapiConsole: boolean
  consoleLevel: string
  consoleTimestamps: boolean
  consoleFollow: boolean
}

export function gamePrefs(s: Settings, game = currentGame()): GamePrefBlock {
  const got = s.games?.[game]
  // SMAPI's builds and console choices are the loader's, stored beside the game blocks.
  const loader = {
    smapiBuilds: s.smapiBuilds || text('smapiBuilds', game),
    showSmapiConsole: on(s.showSmapiConsole, flag('showSmapiConsole', game)),
  }
  const defaults: GamePrefBlock = {
    backupBeforePlay: text('backupBeforePlay', game),
    saveBackupsKept: count('saveBackupsKept', game),
    saveBackupHours: count('saveBackupHours', game),
    saveBackupKeep: count('saveBackupKeep', game),
    updateModsBeforePlayDefault: flag('updateModsBeforePlayDefault', game),
    pauseDownloadsWhilePlaying: flag('pauseDownloadsWhilePlaying', game),
    runsKept: count('runsKept', game),
    consoleLogCap: count('consoleLogCap', game),
    nxmDefaultProfile: text('nxmDefaultProfile', game),
    cosmeticConflicts: text('cosmeticConflicts', game),
    enableRequirements: text('enableRequirements', game),
    disableDependencies: text('disableDependencies', game),
    missingRequirements: text('missingRequirements', game),
    defaultLaunchMethod: text('defaultLaunchMethod', game),
    startFirst: text('startFirst', game),
    consoleLevel: text('consoleLevel', game),
    consoleTimestamps: flag('consoleTimestamps', game),
    consoleFollow: flag('consoleFollow', game),
    ...loader,
  }
  if (!got) {
    return defaults
  }
  return {
    backupBeforePlay: got.backupBeforePlay || defaults.backupBeforePlay,
    saveBackupsKept: got.saveBackupsKept || defaults.saveBackupsKept,
    saveBackupHours: got.saveBackupHours ?? defaults.saveBackupHours,
    saveBackupKeep: got.saveBackupKeep || defaults.saveBackupKeep,
    updateModsBeforePlayDefault: Boolean(got.updateModsBeforePlayDefault),
    pauseDownloadsWhilePlaying: Boolean(got.pauseDownloadsWhilePlaying),
    runsKept: got.runsKept || defaults.runsKept,
    consoleLogCap: got.consoleLogCap || defaults.consoleLogCap,
    nxmDefaultProfile: got.nxmDefaultProfile ?? '',
    cosmeticConflicts: got.cosmeticConflicts || defaults.cosmeticConflicts,
    enableRequirements: got.enableRequirements || defaults.enableRequirements,
    disableDependencies: got.disableDependencies || defaults.disableDependencies,
    missingRequirements: got.missingRequirements || defaults.missingRequirements,
    defaultLaunchMethod: got.defaultLaunchMethod || defaults.defaultLaunchMethod,
    startFirst: got.startFirst || defaults.startFirst,
    consoleLevel: got.consoleLevel || defaults.consoleLevel,
    consoleTimestamps: on(got.consoleTimestamps, defaults.consoleTimestamps),
    consoleFollow: on(got.consoleFollow, defaults.consoleFollow),
    ...loader,
  }
}
