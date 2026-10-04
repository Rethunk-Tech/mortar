import type {
  HistoryDiff,
  HistoryItem,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

export function diffLines(diff: HistoryDiff): string[] {
  return (diff.items ?? []).map((item) => item.detail).filter((line) => line !== '')
}

export function selectedPair(ids: string[]): [string, string] | null {
  const [a, b] = ids
  return ids.length === 2 && a !== undefined && b !== undefined ? [a, b] : null
}

export function itemModKey(item: HistoryItem): string {
  if (item.kind === 'config' && item.file) {
    return `${item.mod}/${item.file}`
  }
  return item.mod
}
