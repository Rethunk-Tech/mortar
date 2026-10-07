import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { listNames } from '../i18n/list.ts'
import { type RedundantItem, sameJobGroups } from './sameJobGroups.ts'
import { useMods } from './store.ts'

export interface RedundantRow {
  key: string
  id: string
  name: string
  reason: string
  // A group of mods doing the same job is one row; Remove then asks which of them goes.
  choices?: { key: string; name: string }[]
  text?: string
}

// Each Redundant row's text, naming the enabled mods that make it redundant; mods doing the same job share one row.
export function useRedundantRows() {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  return (items: RedundantItem[]): RedundantRow[] => {
    const names = new Map<string, string>()
    for (const item of items) {
      names.set(item.key, item.name)
      for (const b of item.by ?? []) {
        names.set(b.key, b.name)
      }
    }
    const rows: RedundantRow[] = []
    const sameJob = items.filter((item) => item.kind === 'sameJob' && !item.covered)
    for (const group of sameJobGroups(sameJob)) {
      const label = (key: string) => {
        const name = names.get(key) ?? key
        const twin = group.keys.some((other) => other !== key && names.get(other) === name)
        const author = mods.find((m) => m.key === key)?.author ?? ''
        return twin && author !== '' ? `${name} (${author})` : name
      }
      const choices = group.keys.map((key) => ({ key, name: label(key) }))
      const list = listNames(
        choices.map((c) => c.name),
        choices.length,
      )
      const { detail } = group
      rows.push({
        key: group.keys.join('|'),
        id: '',
        name: '',
        reason: '',
        choices,
        text:
          choices.length === 2
            ? t`${list} do the same job: both change ${detail}`
            : t`${list} do the same job: all ${choices.length} change ${detail}`,
      })
    }
    for (const item of items.filter((x) => x.kind !== 'sameJob' || x.covered)) {
      const byNames = (item.by ?? []).map((b) => b.name)
      const by = listNames(byNames)
      const count = byNames.length
      const detail = item.detail ?? ''
      let reason = t`Every edit it makes is overwritten by ${by}`
      if (item.kind === 'superseded') {
        reason = t`${plural(count, {
          one: `Replaced by ${by}, which is also enabled`,
          other: `Replaced by ${by}, which are also enabled`,
        })}`
      } else if (item.kind === 'bundled') {
        reason = t`${plural(count, {
          one: `${by} already includes everything it changes`,
          other: `${by} together include everything it changes`,
        })}`
      } else if (item.covered) {
        reason = t`${plural(count, {
          one: `Overlaps with ${by}: both change ${detail}`,
          other: `Overlaps with ${by}: all change ${detail}`,
        })}`
      }
      rows.push({ key: item.key, id: item.id, name: item.name, reason })
    }
    return rows
  }
}
