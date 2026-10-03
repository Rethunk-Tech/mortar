export interface HistoryDiffView {
  a: string
  b: string
  added: { id: string; name: string; version: string; key: string }[]
  removed: { id: string; name: string; version: string; key: string }[]
  versions: { id: string; name: string; old: string; new: string; oldKey: string; newKey: string }[]
  enabled: { id: string; name: string; key: string; old: boolean; new: boolean }[]
  configs: { id: string; name: string; key: string; files: string[] }[]
  items: {
    kind: string
    mod: string
    name: string
    key: string
    oldKey?: string
    newKey?: string
    old?: string
    new?: string
    file?: string
    detail: string
  }[]
}

export function diffLines(diff: HistoryDiffView): string[] {
  return (diff.items ?? []).map((item) => item.detail).filter((line) => line !== '')
}

export function selectedPair(ids: string[]): [string, string] | null {
  if (ids.length !== 2) {
    return null
  }
  return [ids[0], ids[1]]
}

export function itemModKey(item: HistoryDiffView['items'][number]): string {
  if (item.kind === 'config' && item.file) {
    return `${item.mod}/${item.file}`
  }
  return item.mod
}
