import type { GameModPreview } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { willImport } from '../profiles/gameModsFormat.ts'

// Rows that can be picked have a folder to act on; skipped and failed rows only explain themselves.
export function partitionPreview(mods: readonly GameModPreview[] | null | undefined) {
  const rows = mods ?? []
  return {
    pickable: rows.filter((m) => willImport(m.status) && (m.folder ?? '') !== ''),
    blocked: rows.filter((m) => !willImport(m.status)),
  }
}

export const toggled = (off: ReadonlySet<string>, folder: string): Set<string> => {
  const next = new Set(off)
  if (!next.delete(folder)) {
    next.add(folder)
  }
  return next
}

export const selectedFolders = (pickable: readonly GameModPreview[], off: ReadonlySet<string>) =>
  pickable.flatMap((m) => (m.folder === undefined || off.has(m.folder) ? [] : [m.folder]))
