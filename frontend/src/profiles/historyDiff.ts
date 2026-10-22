import { msg } from '@lingui/core/macro'
import type {
  HistoryDiff,
  HistoryItem,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { i18n } from '../i18n/index.ts'

// One changed mod of a history entry in the user's language, from its kind and values; an item of an unknown kind keeps
// the English line Go wrote.
export function itemLabel(item: HistoryItem): string {
  const { name } = item
  switch (item.kind) {
    case 'added':
      return i18n._(msg`Added ${name}`)
    case 'removed':
      return i18n._(msg`Removed ${name}`)
    case 'updated': {
      const from = item.old ?? ''
      const to = item.new ?? ''
      return i18n._(msg`Updated ${name} from ${from} to ${to}`)
    }
    case 'enabled':
      return i18n._(msg`Enabled ${name}`)
    case 'disabled':
      return i18n._(msg`Disabled ${{ names: name }}`)
    case 'config': {
      const fileName = item.file ?? ''
      return i18n._(msg`Changed ${fileName} of ${name}`)
    }
    default:
      return item.detail
  }
}

export function diffLines(diff: HistoryDiff): string[] {
  return (diff.items ?? []).map(itemLabel).filter((line) => line !== '')
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
