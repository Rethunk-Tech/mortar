import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { prefCopy } from '../settings/prefCopy.ts'
export const OVERRIDE_KEYS = [
  'defaultLaunchMethod',
  'showSmapiConsole',
  'backupBeforePlay',
  'saveBackupsKept',
  'updateModsBeforePlayDefault',
  'skipPlayCheck',
  'skipIntro',
  'graphicsApi',
] as const

export type OverrideKey = (typeof OVERRIDE_KEYS)[number]

export const OVERRIDE_VALUES: Record<OverrideKey, string[]> = {
  defaultLaunchMethod: ['steam', 'direct'],
  showSmapiConsole: ['true', 'false'],
  backupBeforePlay: ['changed', 'always', 'never'],
  saveBackupsKept: Array.from({ length: 50 }, (_, i) => String(i + 1)),
  updateModsBeforePlayDefault: ['true', 'false'],
  skipPlayCheck: ['true', 'false'],
  skipIntro: ['true', 'false'],
  // The game's catalog entry supplies the choices.
  graphicsApi: [],
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
