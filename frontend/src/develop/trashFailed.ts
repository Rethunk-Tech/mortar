import { msg } from '@lingui/core/macro'
import { i18n } from '../i18n/index.ts'
import { listNames } from '../i18n/list.ts'
import { toastError } from '../toasts/report.ts'

/** The programs holding a folder open, from the `in_use` error's detail; empty when Mortar could not tell. */
export function inUseHolders(e: unknown): string[] {
  const cause: unknown = e instanceof Error ? e.cause : undefined
  if (typeof cause !== 'object' || cause === null || !('detail' in cause)) {
    return []
  }
  const d = cause.detail
  if (typeof d !== 'object' || d === null || !('holders' in d) || !Array.isArray(d.holders)) {
    return []
  }
  return d.holders.filter((h): h is string => typeof h === 'string')
}

/** Toasts a failed move to the trash; a folder some programs hold open names them, in the player's language. */
export function reportTrashFailed(e: unknown) {
  const holders = inUseHolders(e)
  const programs = listNames(holders, holders.length)
  toastError(i18n._(msg`Could not move the folder to the trash`), e, {
    ...(holders.length > 0 ? { detail: i18n._(msg`In use by ${programs}`) } : {}),
  })
}
