import { saveName } from './saveName.ts'

export function filterAndSortSaves<T extends { farm: string; folder: string; played: number }>(
  fits: readonly T[],
  query: string,
): T[] {
  const needle = query.trim().toLowerCase()
  const rows =
    needle === ''
      ? [...fits]
      : fits.filter((fit) => {
          const farm = saveName(fit).toLowerCase()
          return farm.includes(needle) || fit.folder.toLowerCase().includes(needle)
        })
  rows.sort((a, b) => b.played - a.played)
  return rows
}
