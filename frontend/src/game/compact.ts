export const compactQuery = '(max-width:959.95px)'
// Below this width the sidebar becomes a rail and the hero a one-line header.
export const compact = `@media ${compactQuery}`

export const searchFieldOpen = (narrow: boolean, expanded: boolean, query: string) =>
  !narrow || expanded || query !== ''

export function compactMeta(mods: number, updates: number, problems: number) {
  const parts: { n: number; kind: 'mods' | 'updates' | 'problems' }[] = []
  if (mods > 0) {
    parts.push({ n: mods, kind: 'mods' })
  }
  if (updates > 0) {
    parts.push({ n: updates, kind: 'updates' })
  }
  if (problems > 0) {
    parts.push({ n: problems, kind: 'problems' })
  }
  return parts
}

export function saveFits(fits: { missing?: unknown[] | null }[]) {
  return {
    fitting: fits.filter((f) => (f.missing ?? []).length === 0).length,
    total: fits.length,
  }
}
