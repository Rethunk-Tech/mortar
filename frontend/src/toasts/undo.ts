import type { ToastAction, ToastInput } from './store.ts'

export const MISSING_STORE_PREFIX = 'missing from the store: '

export function pushUndoToast(
  push: (toast: ToastInput) => number,
  title: string,
  undoLabel: string,
  undo: () => unknown,
): number {
  const action: ToastAction = { label: undoLabel, run: undo }
  return push({ kind: 'success', title, action })
}

export interface UndoEntry {
  key: string
  previousKey?: string
  extraStoreKeys?: string[] | null
  previousExtraStoreKeys?: string[] | null
  pinned?: boolean
  skipVersion?: string
  tags?: string[] | null
  categoryOverride?: string
  mods?: { name: string }[] | null
  source: {
    kind: string
    name?: string
    version?: string
    modId?: number
    fileId?: number
    repo?: string
    tag?: string
    asset?: string
  }
}

export interface EntryFieldsPayload {
  key: string
  pinned: boolean
  skipVersion: string
  tags: string[]
  categoryOverride: string
}

export interface RevertWant {
  kind: 'install'
  name: string
  version: string
  modId?: number
  fileId?: number
  repo?: string
  tag?: string
  asset?: string
}

export function parseMissingStoreList(message: string): string[] {
  const at = message.indexOf(MISSING_STORE_PREFIX)
  if (at < 0) {
    return []
  }
  return message
    .slice(at + MISSING_STORE_PREFIX.length)
    .split(', ')
    .filter(Boolean)
}

export function entriesMatchingMissing(entries: UndoEntry[], missing: string[]): UndoEntry[] {
  const want = new Set(missing)
  return entries.filter((entry) => {
    if (want.has(entry.key) || (entry.previousKey !== undefined && want.has(entry.previousKey))) {
      return true
    }
    for (const key of [...(entry.extraStoreKeys ?? []), ...(entry.previousExtraStoreKeys ?? [])]) {
      if (want.has(key)) {
        return true
      }
    }
    return (entry.mods ?? []).some((mod) => want.has(mod.name))
  })
}

export function missingModNames(entries: UndoEntry[]): string[] {
  const names: string[] = []
  for (const entry of entries) {
    const label =
      (entry.mods ?? [])
        .map((mod) => mod.name)
        .filter(Boolean)
        .join(', ') || entry.key
    if (!names.includes(label)) {
      names.push(label)
    }
  }
  return names
}

export function downloadWantsForEntries(entries: UndoEntry[]): RevertWant[] {
  const wants: RevertWant[] = []
  for (const entry of entries) {
    const { source } = entry
    if (source.kind === 'nexus' && source.modId && source.fileId) {
      wants.push({
        kind: 'install',
        name: source.name ?? '',
        version: source.version ?? '',
        modId: source.modId,
        fileId: source.fileId,
      })
    } else if (source.kind === 'github' && source.repo) {
      wants.push({
        kind: 'install',
        name: source.name ?? '',
        version: source.version ?? '',
        repo: source.repo,
        ...(source.tag === undefined ? {} : { tag: source.tag }),
        ...(source.asset === undefined ? {} : { asset: source.asset }),
      })
    }
  }
  return wants
}

export function entryFieldsOf(entries: UndoEntry[], keys: string[]): EntryFieldsPayload[] {
  const want = new Set(keys)
  return entries
    .filter((entry) => want.has(entry.key))
    .map((entry) => ({
      key: entry.key,
      pinned: entry.pinned ?? false,
      skipVersion: entry.skipVersion ?? '',
      tags: [...(entry.tags ?? [])],
      categoryOverride: entry.categoryOverride ?? '',
    }))
}

export function entriesForKeys<T extends { key: string }>(entries: T[], keys: string[]): T[] {
  const want = new Set(keys)
  return entries.filter((entry) => want.has(entry.key))
}

export function undoRevertTarget(events: { id: string }[]): string {
  return events[0]?.id ?? ''
}

export function fieldsStillUndoable(
  entries: UndoEntry[] | undefined,
  prev: EntryFieldsPayload[],
): boolean {
  if (!entries) {
    return false
  }
  return prev.some((field) => {
    const entry = entries.find((item) => item.key === field.key)
    if (!entry) {
      return false
    }
    if ((entry.pinned ?? false) !== field.pinned) {
      return true
    }
    if ((entry.skipVersion ?? '') !== field.skipVersion) {
      return true
    }
    if ((entry.categoryOverride ?? '') !== field.categoryOverride) {
      return true
    }
    const now = [...(entry.tags ?? [])].toSorted()
    const was = [...field.tags].toSorted()
    return now.length !== was.length || now.some((tag, i) => tag !== was[i])
  })
}
