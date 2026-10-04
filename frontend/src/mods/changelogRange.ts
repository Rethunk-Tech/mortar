import type { Changelog } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexus/models.ts'
import { isNewer } from './nexusFormat.ts'

const riskyChangelogPhrases = [
  'breaking',
  'incompatible',
  'requires',
  'now needs',
  'new save',
  'start a new',
  'not save compatible',
  'remove before updating',
] as const

export function changelogNoteIsRisky(note: string): boolean {
  const lower = note.toLowerCase()
  return riskyChangelogPhrases.some((phrase) => lower.includes(phrase))
}

export function changelogsHaveRiskyNotes(logs: Changelog[]): boolean {
  for (const entry of logs) {
    for (const note of entry.notes ?? []) {
      if (changelogNoteIsRisky(note)) {
        return true
      }
    }
  }
  return false
}

// Versions newer than installed and not newer than latest (latest included, installed excluded).
export const changelogsBetween = (
  logs: Changelog[] | null | undefined,
  installed: string,
  latest: string,
): Changelog[] =>
  (logs ?? []).filter((c) => isNewer(c.version, installed) && !isNewer(c.version, latest))
