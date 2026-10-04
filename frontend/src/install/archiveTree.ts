import type {
  PreviewEntry,
  PreviewManifest,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/archive/models.ts'

interface TreeRow {
  path: string
  name: string
  depth: number
  isDir: boolean
  size: number
  manifest?: PreviewManifest
}

interface TreeGroup {
  // The archive's top-level folder; '' collects files that sit at the archive's root.
  folder: string
  rows: TreeRow[]
  size: number
}

const segments = (path: string) => path.replaceAll('\\', '/').split('/').filter(Boolean)

function compareSegments(a: string[], b: string[]): number {
  for (let i = 0; i < Math.min(a.length, b.length); i++) {
    const x = a[i] ?? ''
    const y = b[i] ?? ''
    if (x !== y) {
      return x < y ? -1 : 1
    }
  }
  return a.length - b.length
}

/** An archive listing as indented rows grouped by top folder, with folders the listing leaves implicit filled in. */
export function groupEntries(
  entries: readonly PreviewEntry[],
  manifests: readonly PreviewManifest[],
): TreeGroup[] {
  const rows = new Map<string, TreeRow>()
  for (const entry of entries) {
    const parts = segments(entry.path)
    for (let i = 1; i < parts.length; i++) {
      const dir = parts.slice(0, i).join('/')
      rows.set(
        dir,
        rows.get(dir) ?? {
          path: dir,
          name: parts[i - 1] ?? '',
          depth: i - 1,
          isDir: true,
          size: 0,
        },
      )
    }
    const path = parts.join('/')
    if (path) {
      rows.set(path, {
        path,
        name: parts.at(-1) ?? '',
        depth: parts.length - 1,
        isDir: entry.isDir,
        size: entry.size,
      })
    }
  }
  for (const manifest of manifests) {
    const row = rows.get(segments(manifest.folder).join('/'))
    if (row) {
      row.manifest = manifest
    }
  }
  const groups = new Map<string, TreeGroup>()
  const sorted = [...rows.values()].sort((a, b) =>
    compareSegments(segments(a.path), segments(b.path)),
  )
  for (const row of sorted) {
    const top = row.depth === 0 && !row.isDir ? '' : (segments(row.path)[0] ?? '')
    const group = groups.get(top) ?? { folder: top, rows: [], size: 0 }
    group.rows.push(row)
    group.size += row.isDir ? 0 : row.size
    groups.set(top, group)
  }
  return [...groups.values()]
}

export type { TreeGroup, TreeRow }
