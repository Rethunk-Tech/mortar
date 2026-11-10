import type { ListColumnId, ListRow } from './listColumns.ts'

const FIGURE_COLUMNS: readonly ListColumnId[] = [
  'latest',
  'released',
  'updated',
  'pluginVersion',
  'kind',
  'id',
  'endorsements',
  'downloads',
]

// The only columns that read a row field that data arriving later fills in; with none of them shown, that field
// changing leaves the row as it was (the list never reads a row's details).
const FIELD_COLUMNS: Partial<Record<keyof ListRow, readonly ListColumnId[]>> = {
  size: ['size'],
  startupMs: ['startup'],
  facts: FIGURE_COLUMNS,
  figures: FIGURE_COLUMNS,
  details: [],
}

// List rows are rebuilt whenever the mods, the profile or the problems change, so a row compares the fields its
// shown columns read, not identity; tags and figures are built fresh each time and compare by content.
export function sameRowFields(a: ListRow, b: ListRow, cols: readonly ListColumnId[]) {
  const keys = new Set([...Object.keys(a), ...Object.keys(b)] as (keyof ListRow)[])
  return [...keys].every((k) => !isRead(k, cols) || sameField(k, a, b))
}

function isRead(k: keyof ListRow, cols: readonly ListColumnId[]) {
  const readers = FIELD_COLUMNS[k]
  return readers === undefined || readers.some((c) => cols.includes(c))
}

function sameField(k: keyof ListRow, a: ListRow, b: ListRow) {
  // The pin shows through VersionPin and PinBadge, which read the store themselves.
  if (k === 'pinned') {
    return true
  }
  if (k === 'tags') {
    return a.tags.join('\0') === b.tags.join('\0')
  }
  if (k === 'figures' && a.figures && b.figures) {
    const [x, y] = [a.figures, b.figures]
    return (
      x.latest === y.latest &&
      x.released === y.released &&
      x.updated === y.updated &&
      x.rating === y.rating &&
      x.downloads === y.downloads
    )
  }
  return a[k] === b[k]
}
