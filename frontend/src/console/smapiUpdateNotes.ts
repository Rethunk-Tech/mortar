import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import type { UpdatesResult } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'

/** Why Mortar's update list differs from one line of SMAPI's "You can update" alert. */
type UpdateNoteKind =
  | 'current'
  | 'skipped'
  | 'pinned'
  | 'ignored'
  | 'source'
  | 'prerelease'
  | 'unofficial'
  | 'listed'
  | 'elsewhere'

interface UpdateNote {
  kind: UpdateNoteKind
  have?: string
}

// SMAPI writes each update as "   Name 1.2.3: https://… (you have 1.2.0)".
const LINE = /^\s*(.+?) (\S+): (https?:\/\/\S+)(?: \(you have ([^)]+)\))?\s*$/

const OWN_SITES = /nexusmods\.com|github\.com/i

const sameName = (a: string, b: string) => a.trim().toLowerCase() === b.trim().toLowerCase()

function noteFor(
  result: UpdatesResult,
  name: string,
  version: string,
  url: string,
): UpdateNote | undefined {
  const held = (result.held ?? []).find((h) => sameName(h.name, name) && h.version === version)
  if (held) {
    return { kind: held.reason as UpdateNoteKind, have: held.have }
  }
  const listed = (result.updates ?? []).find((u) => sameName(u.name, name) && u.version === version)
  if (listed) {
    return { kind: listed.nexusId > 0 || listed.githubRepo !== '' ? 'listed' : 'elsewhere' }
  }
  return OWN_SITES.test(url) ? undefined : { kind: 'elsewhere' }
}

/** Notes for SMAPI's update alert rows, keyed by entry seq. */
function smapiUpdateNotes(
  rows: readonly Entry[],
  result: UpdatesResult | null,
): Map<number, UpdateNote> {
  const notes = new Map<number, UpdateNote>()
  if (!result) {
    return notes
  }
  for (const row of rows) {
    const m = row.level === 'ALERT' && row.mod === 'SMAPI' ? LINE.exec(row.message) : null
    if (m) {
      const [, name = '', version = '', url = ''] = m
      const note = noteFor(result, name, version, url)
      if (note) {
        notes.set(row.seq, note)
      }
    }
  }
  return notes
}

export { smapiUpdateNotes, type UpdateNote }
