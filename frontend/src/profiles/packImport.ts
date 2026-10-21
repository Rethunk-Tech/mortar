import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import type { Package } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/models.ts'
import { listNames } from '../i18n/list.ts'

/** Whether a game's catalog entry lists Thunderstore among its mod sources. */
export function hasThunderstore(game: { sources: string[] | null } | null | undefined): boolean {
  return game?.sources?.includes('thunderstore') === true
}

/** One package of a pack as a list line: its name and version, with `off` when the pack disabled it. */
export function packageLine(p: Package, off: string): string {
  const line = p.version ? `${p.native} ${p.version}` : p.native
  return p.disabled ? `${line} · ${off}` : line
}

/** What an import left undone: packages from other sources, and files the player's profile already held with other contents. */
export function importNotes(i18n: I18n, unsupported: string[], kept: string[], skipped: string[] = []): string {
  return [
    unsupported.length > 0 ? i18n._(msg`Not from Thunderstore: ${listNames(unsupported)}`) : '',
    skipped.length > 0 ? i18n._(msg`Skipped, not on Thunderstore: ${listNames(skipped)}`) : '',
    kept.length > 0 ? i18n._(msg`Kept your own copies of: ${listNames(kept)}`) : '',
  ]
    .filter(Boolean)
    .join('. ')
}
