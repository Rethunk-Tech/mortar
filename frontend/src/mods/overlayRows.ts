import type {
  Entry,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import type { OverlayRow, VirtualRow } from './virtualRows.ts'

// The Nexus file name an optional file was installed from, or its store key.
const overlayLabel = (e: Pick<Entry, 'key' | 'source'>) => e.source.name || e.key

function anyModOn(base: Entry | undefined) {
  return (base?.mods ?? []).some((m) => !(base?.disabled ?? []).includes(m.uniqueId))
}

/** Each main entry's optional files, in the order they are laid over it. */
export function overlaysByBase(
  profile: Pick<Profile, 'entries'> | null | undefined,
): Map<string, OverlayRow[]> {
  const entries = profile?.entries ?? []
  const out = new Map<string, OverlayRow[]>()
  for (const e of entries.filter((x) => (x.overlayOf ?? '') !== '')) {
    const baseKey = e.overlayOf ?? ''
    const row: OverlayRow = {
      key: e.key,
      baseKey,
      label: overlayLabel(e),
      enabled: !(e.overlayOff ?? false),
      baseEnabled: anyModOn(entries.find((b) => b.key === baseKey)),
    }
    out.set(baseKey, [...(out.get(baseKey) ?? []), row])
  }
  return out
}

/** Puts each main entry's optional files right after its first visible row, once. */
export function nestOverlays<T>(
  rows: readonly VirtualRow<T>[],
  keyOf: (item: T) => string,
  byBase: ReadonlyMap<string, readonly OverlayRow[]>,
): VirtualRow<T>[] {
  const out: VirtualRow<T>[] = []
  const done = new Set<string>()
  for (const row of rows) {
    out.push(row)
    const key = row.kind === 'row' ? keyOf(row.item) : ''
    const nested = done.has(key) ? undefined : byBase.get(key)
    if (nested) {
      done.add(key)
      for (const overlay of nested) {
        out.push({ kind: 'overlay', key: `o:${overlay.key}`, groupKey: row.groupKey, overlay })
      }
    }
  }
  return out
}

export { overlayLabel }
