import { create } from 'zustand'
import type { File } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexus/models.ts'
import type { Update } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type {
  OverlayFileSet,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { Want } from '../queue/actions.ts'
import { parseBBCode } from './bbcode.ts'

const OPTIONAL_CATEGORIES = new Set(['OPTIONAL', 'MISCELLANEOUS'])
const RETIRED_CATEGORIES = new Set(['OLD_VERSION', 'ARCHIVED'])
const ARCHIVE_EXT = /\.(zip|rar|7z)$/i
const VERSION_SUFFIX = /(?:[\s._-]+v?\d+(?:[._-]\d+)+|[\s._-]+v?\d+)$/i

type Entries = Pick<Profile, 'entries'> | null | undefined

/** A mod page's current Optional and Miscellaneous files, as Nexus lists them. */
const nexusOptionalFiles = (files: readonly File[]) =>
  files.filter((f) => OPTIONAL_CATEGORIES.has(f.category.toUpperCase()))

/** The Nexus file ids of modId the profile holds, so the page's files can be marked installed. */
const installedFileIds = (profile: Entries, modId: number) =>
  new Set(
    (profile?.entries ?? [])
      .filter((e) => e.source.kind === 'nexus' && e.source.modId === modId)
      .map((e) => e.source.fileId ?? 0),
  )

/** A Nexus file description as one line of plain text. */
const plainDescription = (bbcode: string) =>
  parseBBCode(bbcode)
    .flatMap((b) => b.runs.map((r) => r.text))
    .join(' ')
    .replace(/\s+/g, ' ')
    .trim()

const fileStem = (name: string) => {
  let stem = name.trim().replace(ARCHIVE_EXT, '').trim()
  for (let next = stem.replace(VERSION_SUFFIX, '').trim(); next !== stem; ) {
    stem = next
    next = stem.replace(VERSION_SUFFIX, '').trim()
  }
  return stem.toLowerCase().replace(/\s+/g, ' ')
}

// Two files are versions of one download when the author's file_updates chain links them, they share Nexus's
// display name, or (with no names) their archive names match once versions are stripped.
const sameGroup = (a: File, b: File) => {
  if (a.replacedBy === b.fileId || b.replacedBy === a.fileId) {
    return true
  }
  if (a.name && b.name) {
    return a.name.trim().toLowerCase() === b.name.trim().toLowerCase()
  }
  const stem = fileStem(a.fileName)
  return stem !== '' && stem === fileStem(b.fileName)
}

/** The newest listed version of file fileId, following the author's file_updates chain; undefined when it is current. */
function newerVersion(files: readonly File[], fileId: number): File | undefined {
  const byId = new Map(files.map((f) => [f.fileId, f]))
  let cur = byId.get(fileId)
  if (!cur) {
    return undefined
  }
  const seen = new Set<number>()
  for (
    let next = byId.get(cur.replacedBy);
    next && !seen.has(cur.fileId);
    next = byId.get(cur.replacedBy)
  ) {
    seen.add(cur.fileId)
    cur = next
  }
  for (const f of files) {
    if (
      f.fileId > cur.fileId &&
      !RETIRED_CATEGORIES.has(f.category.toUpperCase()) &&
      sameGroup(f, cur)
    ) {
      cur = f
    }
  }
  return cur.fileId === fileId ? undefined : cur
}

/** Queue requests for newer versions of the optional files laid over the entry an update replaces. */
function optionalUpdateWants(profile: Entries, update: Update, files: readonly File[]): Want[] {
  const wants: Want[] = []
  const overlays = (profile?.entries ?? []).filter(
    (e) =>
      e.overlayOf === update.key &&
      e.source.kind === 'nexus' &&
      (e.source.modId ?? 0) === update.nexusId &&
      Boolean(e.source.fileId),
  )
  for (const e of overlays) {
    const { modId = 0, fileId = 0 } = e.source
    const newer = newerVersion(files, fileId)
    if (newer) {
      wants.push({
        kind: 'update',
        modId,
        fileId: newer.fileId,
        name: e.source.modName || update.name,
        fileName: newer.fileName,
        version: newer.version,
        currentKey: `nexus-${modId}-${fileId}`,
      })
    }
  }
  return wants
}

/** Update keys whose optional files the user left out of the update. */
const useOptionalSkips = create<{
  skipped: Record<string, boolean>
  setSkipped: (key: string, on: boolean) => void
}>((set) => ({
  skipped: {},
  setSkipped: (key, on) => set((s) => ({ skipped: { ...s.skipped, [key]: on } })),
}))

/** Groups of optional files that replace some of the same files, in profile order; each lists two or more keys. */
function alternativeGroups(sets: readonly OverlayFileSet[]): string[][] {
  const groups: string[][] = []
  const placed = new Set<string>()
  for (const set of sets) {
    if (placed.has(set.key) || (set.alternatives ?? []).length === 0) {
      continue
    }
    const group: string[] = []
    const todo = [set.key]
    while (todo.length > 0) {
      const key = todo.shift() ?? ''
      if (placed.has(key)) {
        continue
      }
      placed.add(key)
      group.push(key)
      todo.push(...(sets.find((s) => s.key === key)?.alternatives ?? []))
    }
    groups.push(sets.map((s) => s.key).filter((k) => group.includes(k)))
  }
  return groups
}

export {
  alternativeGroups,
  installedFileIds,
  newerVersion,
  nexusOptionalFiles,
  optionalUpdateWants,
  plainDescription,
  useOptionalSkips,
}
