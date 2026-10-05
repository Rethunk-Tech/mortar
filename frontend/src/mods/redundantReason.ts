import { useLingui } from '@lingui/react/macro'
import { listNames } from '../i18n/list.ts'
import { useMods } from './store.ts'

interface RedundantItem {
  kind: string
  key: string
  id: string
  name: string
  by: { key: string; name: string }[] | null
  detail?: string
  covered?: boolean
}

export interface RedundantRow {
  key: string
  id: string
  name: string
  reason: string
  // A group of mods doing the same job is one row; Remove then asks which of them goes.
  choices?: { key: string; name: string }[]
  text?: string
}

// sameJobGroups joins every similar-job pair into the set of mods that all do that job, in first-seen order.
export function sameJobGroups(items: RedundantItem[]): { keys: string[]; detail: string }[] {
  const parent = new Map<string, string>()
  const find = (k: string): string => {
    const p = parent.get(k) ?? k
    if (p === k) {
      return k
    }
    const root = find(p)
    parent.set(k, root)
    return root
  }
  const order: string[] = []
  const detail = new Map<string, string>()
  for (const item of items) {
    for (const k of [item.key, ...(item.by ?? []).map((b) => b.key)]) {
      if (!parent.has(k)) {
        parent.set(k, k)
        order.push(k)
      }
    }
    for (const b of item.by ?? []) {
      parent.set(find(b.key), find(item.key))
    }
  }
  for (const item of items) {
    const root = find(item.key)
    if (!detail.has(root) && item.detail) {
      detail.set(root, item.detail)
    }
  }
  const groups = new Map<string, string[]>()
  for (const k of order) {
    const root = find(k)
    groups.set(root, [...(groups.get(root) ?? []), k])
  }
  return [...groups.entries()].map(([root, keys]) => ({ keys, detail: detail.get(root) ?? '' }))
}

// Each Redundant row's text, naming the enabled mods that make it redundant; mods doing the same job share one row.
export function useRedundantRows() {
  const { t, i18n } = useLingui()
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
      const list = new Intl.ListFormat(i18n.locale, { type: 'conjunction' }).format(
        choices.map((c) => c.name),
      )
      const { detail } = group
      rows.push({
        key: group.keys.join('|'),
        id: '',
        name: '',
        reason: '',
        choices,
        text: t`${list} do the same job: all change ${detail}`,
      })
    }
    for (const item of items.filter((x) => x.kind !== 'sameJob' || x.covered)) {
      const by = listNames((item.by ?? []).map((b) => b.name))
      const detail = item.detail ?? ''
      let reason = t`Every edit it makes is overwritten by ${by}`
      if (item.kind === 'superseded') {
        reason = t`Replaced by ${by}, which is also enabled`
      } else if (item.covered) {
        reason = t`Everything it changes, ${by} also changes: ${detail}`
      }
      rows.push({ key: item.key, id: item.id, name: item.name, reason })
    }
    return rows
  }
}
