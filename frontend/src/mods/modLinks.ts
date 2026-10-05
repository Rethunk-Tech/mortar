import type { Row } from './problemGroups.ts'

export interface ModLink {
  name: string
  key: string
  id: string
}

// modLinksOf is the mods a problem row names, in the order its sentence is likely to mention them.
export function modLinksOf(row: Row): ModLink[] {
  switch (row.kind) {
    case 'broken':
      return [{ name: row.broken.name, key: row.broken.key, id: row.broken.id }]
    case 'runError':
      return [{ name: row.runError.name, key: row.runError.key, id: row.runError.id }]
    case 'setting':
      return [{ name: row.setting.name, key: row.setting.key, id: row.setting.id }]
    case 'duplicate': {
      const key = row.duplicate.copies?.[0]?.key
      return key === undefined ? [] : [{ name: row.duplicate.name, key, id: row.duplicate.id }]
    }
    case 'asset': {
      const { names, keys, packIds } = row.asset
      return (names ?? []).flatMap((name, i) => {
        const key = keys?.[i]
        const id = packIds?.[i]
        return key === undefined || id === undefined ? [] : [{ name, key, id }]
      })
    }
    default:
      return []
  }
}
