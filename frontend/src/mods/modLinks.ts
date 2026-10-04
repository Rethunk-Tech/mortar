import type { Row } from './problemGroups.ts'

export interface ModLink {
  name: string
  key: string
  uniqueId: string
}

// modLinksOf is the mods a problem row names, in the order its sentence is likely to mention them.
export function modLinksOf(row: Row): ModLink[] {
  switch (row.kind) {
    case 'broken':
      return [{ name: row.broken.name, key: row.broken.key, uniqueId: row.broken.uniqueId }]
    case 'runError':
      return [{ name: row.runError.name, key: row.runError.key, uniqueId: row.runError.uniqueId }]
    case 'setting':
      return [{ name: row.setting.name, key: row.setting.key, uniqueId: row.setting.uniqueId }]
    case 'duplicate': {
      const key = row.duplicate.copies?.[0]?.key
      return key === undefined
        ? []
        : [{ name: row.duplicate.name, key, uniqueId: row.duplicate.uniqueId }]
    }
    case 'asset': {
      const { names, keys, packIds } = row.asset
      return (names ?? []).flatMap((name, i) => {
        const key = keys?.[i]
        const uniqueId = packIds?.[i]
        return key === undefined || uniqueId === undefined ? [] : [{ name, key, uniqueId }]
      })
    }
    default:
      return []
  }
}
