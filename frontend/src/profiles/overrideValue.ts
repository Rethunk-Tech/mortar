import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { prefCopy } from '../settings/prefCopy.ts'
export const OVERRIDE_KEYS = [
  'defaultLaunchMethod',
  'instances',
  'delaySeconds',
  'showSmapiConsole',
  'backupBeforePlay',
  'saveBackupsKept',
  'updateModsBeforePlayDefault',
  'skipPlayCheck',
  'pauseDownloadsWhilePlaying',
  'skipIntro',
  'graphicsApi',
  'protonVersion',
  'gameSettingsMode',
  'cacheClearing',
  'dependencyVersions',
  'saveBackupHours',
  'saveBackupKeep',
  'runsKept',
  'consoleLogCap',
  'consoleLevel',
  'consoleTimestamps',
  'consoleFollow',
  'cosmeticConflicts',
  'conflictScanDepth',
  'enableRequirements',
  'missingRequirements',
] as const

export type OverrideKey = (typeof OVERRIDE_KEYS)[number]

export const OVERRIDE_VALUES: Record<OverrideKey, string[]> = {
  defaultLaunchMethod: ['steam', 'direct'],
  instances: ['1', '2', '3', '4', '5', '6', '7', '8'],
  delaySeconds: Array.from({ length: 60 }, (_, i) => String(i + 1)),
  showSmapiConsole: ['true', 'false'],
  backupBeforePlay: ['changed', 'always', 'never'],
  saveBackupsKept: Array.from({ length: 50 }, (_, i) => String(i + 1)),
  updateModsBeforePlayDefault: ['true', 'false'],
  skipPlayCheck: ['true', 'false'],
  pauseDownloadsWhilePlaying: ['true', 'false'],
  skipIntro: ['true', 'false'],
  // The game's catalog entry supplies the choices.
  graphicsApi: [],
  // This computer's Proton builds supply the choices.
  protonVersion: [],
  gameSettingsMode: ['edit', 'warn'],
  cacheClearing: ['auto', 'off'],
  dependencyVersions: ['newest', 'exact'],
  saveBackupHours: ['0', '1', '2', '3', '4', '6', '8', '12', '24', '48', '72', '168'],
  saveBackupKeep: Array.from({ length: 50 }, (_, i) => String(i + 1)),
  runsKept: ['5', '10', '20', '50', '100'],
  consoleLogCap: ['1000', '5000', '10000', '20000', '50000', '100000'],
  consoleLevel: ['trace', 'debug', 'info', 'warn', 'error'],
  consoleTimestamps: ['true', 'false'],
  consoleFollow: ['true', 'false'],
  cosmeticConflicts: ['collapsed', 'expanded', 'hidden'],
  conflictScanDepth: ['full', 'skipImages'],
  enableRequirements: ['always', 'ask', 'never'],
  missingRequirements: ['ask', 'autodownload', 'never'],
}

export type OverrideChoice = { useGame: true } | { useGame: false; value: string }

export function choiceFromOverride(stored: string | undefined): OverrideChoice {
  if (stored === undefined || stored === '') {
    return { useGame: true }
  }
  return { useGame: false, value: stored }
}

export function rowValue(choice: OverrideChoice): string | undefined {
  if (choice.useGame) {
    return undefined
  }
  return choice.value
}

export function applyRow(
  overrides: Record<string, string>,
  key: string,
  choice: OverrideChoice,
): Record<string, string> {
  const next = { ...overrides }
  const value = rowValue(choice)
  if (value === undefined) {
    delete next[key]
  } else {
    next[key] = value
  }
  return next
}

export function resolveOverride(
  key: string,
  gameValue: string,
  overrides?: Record<string, string> | null,
): string {
  if (overrides && Object.hasOwn(overrides, key)) {
    return overrides[key] ?? gameValue
  }
  return gameValue
}

export function foldedOverrides(profile: Pick<Profile, 'overrides'>): Record<string, string> {
  const out: Record<string, string> = {}
  for (const [key, value] of Object.entries(profile.overrides ?? {})) {
    if (value !== undefined) {
      out[key] = value
    }
  }
  return out
}

export function overrideChoiceLabel(key: OverrideKey, value: string, i18n: I18n): string {
  const hit = prefCopy(i18n, key).options?.find((option) => option.value === value)
  if (hit) {
    return hit.label
  }
  if (value === 'true') {
    return i18n._(msg`On`)
  }
  if (value === 'false') {
    return i18n._(msg`Off`)
  }
  return value
}
