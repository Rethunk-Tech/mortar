// Fails when two msgids in the English catalog say the same thing: they differ only in case, punctuation or placeholder
// names. "…" and "?" stay significant, since a control that opens a dialog and the dialog's title are two messages.
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const CATALOG = resolve(import.meta.dir, '../frontend/src/locales/en/messages.po')

function msgids(po: string): string[] {
  const ids: string[] = []
  for (const block of po.split('\n\n')) {
    const match = /^msgid ((?:".*"\n?)+)/m.exec(block)
    if (!match?.[1]) {
      continue
    }
    const id = [...match[1].matchAll(/"(.*)"/g)].map((m) => m[1]).join('')
    if (id !== '') {
      ids.push(id.replaceAll('\\"', '"'))
    }
  }
  return ids
}

function normalise(id: string): string | null {
  const flat = id
    .replace(/\{[\w.]+(?=, (?:plural|select|selectordinal),)/g, '{')
    .replace(/\{[\w.]+\}/g, '{}')
    .toLowerCase()
  // "{0} · {1}" and "{0} of {1}" are layout, not wording: nothing to say twice.
  if (flat.replace(/\{\}/g, '').replace(/[^\p{L}]/gu, '').length < 4) {
    return null
  }
  return flat
    .replace(/[.,:;!'"‘’“”]/g, '')
    .replace(/\s+/g, ' ')
    .trim()
}

function duplicates(ids: string[]): string[][] {
  const groups = new Map<string, string[]>()
  for (const id of ids) {
    const key = normalise(id)
    if (key === null) {
      continue
    }
    groups.set(key, [...(groups.get(key) ?? []), id])
  }
  return [...groups.values()].filter((g) => g.length > 1)
}

if (import.meta.main) {
  const dupes = duplicates(msgids(readFileSync(CATALOG, 'utf8')))
  for (const group of dupes) {
    console.error(
      `i18n: one message written ${group.length} ways: ${group.map((g) => JSON.stringify(g)).join(' | ')}`,
    )
  }
  if (dupes.length > 0) {
    console.error(
      `i18n: ${dupes.length} duplicate messages; use one wording (docs/gui-design.md, Copy)`,
    )
    process.exit(1)
  }
}

export { duplicates }
