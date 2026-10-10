import type {
  Entry,
  Source,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { idKey } from './dependents.ts'

/** The archive a folder game split into one entry per file, shown as one group. */
export interface ArchiveGroup {
  kind: 'archive'
  item: string
  source: Source
  /** The per-file entries, in profile order. */
  entries: Entry[]
  files: number
  /** Files whose every component is switched off. */
  off: number
  /** The entry holding the archive's Tray files, when it has any. */
  tray?: Entry | undefined
}

/** An entry shown as its own row, with the Tray entry of its archive when it has one. */
export interface SingleRow {
  kind: 'single'
  entry: Entry
  tray?: Entry | undefined
}

export type FolderRow = ArchiveGroup | SingleRow

function isTrayEntry(e: Entry): boolean {
  return e.tray === true
}

function isOff(e: Entry): boolean {
  const mods = e.mods ?? []
  const off = new Set((e.disabled ?? []).map(idKey))
  return mods.length > 0 && mods.every((m) => off.has(idKey(m.id)))
}

/**
 * The Mods list's rows for a profile of a folder game: the per-file entries of one archive (the entries sharing
 * `item`) form one group, a kept-whole archive and any other entry stay single rows, and an archive's Tray entry rides
 * with its group or row. Rows keep the profile's order, a group standing where its first file does.
 */
export function folderRows(entries: readonly Entry[]): FolderRow[] {
  const trays = new Map<string, Entry>()
  for (const e of entries) {
    if (isTrayEntry(e)) {
      trays.set(e.item ?? '', e)
    }
  }
  const owners = new Set<string>()
  for (const e of entries) {
    if (!isTrayEntry(e)) {
      owners.add(e.item || e.key)
    }
  }
  const groups = new Map<string, ArchiveGroup>()
  const rows: FolderRow[] = []
  for (const e of entries) {
    if (isTrayEntry(e)) {
      if (!owners.has(e.item ?? '')) {
        rows.push({ kind: 'single', entry: e })
      }
      continue
    }
    const item = e.item ?? ''
    if (item === '') {
      rows.push({ kind: 'single', entry: e, tray: trays.get(e.key) })
      continue
    }
    const g: ArchiveGroup = groups.get(item) ?? {
      kind: 'archive',
      item,
      source: e.source,
      entries: [],
      files: 0,
      off: 0,
      tray: trays.get(item),
    }
    if (!groups.has(item)) {
      groups.set(item, g)
      rows.push(g)
    }
    g.entries.push(e)
    g.files += 1
    if (isOff(e)) {
      g.off += 1
    }
  }
  return rows
}
