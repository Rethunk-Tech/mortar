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

// A save that records no mod list (Lethal Company's) can neither fit nor miss, so it counts only toward the total.
export function saveFits(fits: { missing?: unknown[] | null; unrecorded?: boolean }[]) {
  const recorded = fits.filter((f) => !f.unrecorded)
  return {
    fitting: recorded.filter((f) => (f.missing ?? []).length === 0).length,
    recorded: recorded.length,
    total: fits.length,
  }
}
