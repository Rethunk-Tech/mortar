import { expect, mock, test } from 'bun:test'
import { HistoryChange } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

mock.module('@lingui/core/macro', () => ({
  msg: (parts: TemplateStringsArray, ...values: unknown[]) =>
    String.raw(
      { raw: parts },
      ...values.map((v) => (v !== null && typeof v === 'object' ? Object.values(v)[0] : v)),
    ),
  plural: (n: number, forms: { one: string; other: string }) =>
    (n === 1 ? forms.one : forms.other).replace('#', String(n)),
}))

const { historyDays, historyKind, historySummary } = await import('./historyTimeline.ts')
const { sentence } = await import('../i18n/sentence.ts')

const NOW = new Date(2026, 9, 7, 15, 0)
const at = (day: number, hour: number) => new Date(2026, 9, day, hour).toISOString()

test('historyDays groups newest-first events by local day', () => {
  const days = historyDays(
    [{ at: at(7, 14) }, { at: at(7, 9) }, { at: at(6, 20) }, { at: at(2, 8) }, { at: at(2, 7) }],
    NOW,
  )
  expect(days.map((d) => [d.label, d.events.length])).toEqual([
    ['today', 2],
    ['yesterday', 1],
    ['date', 2],
  ])
  expect(historyDays([], NOW)).toEqual([])
})

test('historySummary counts a multi-mod change by kind and words the rest as the event does', () => {
  expect(historySummary({ change: HistoryChange.ChangeMods, added: 2 })).toBe('Added 2 mods')
  expect(
    historySummary({ change: HistoryChange.ChangeMods, added: 1, removed: 3, updated: 1 }),
  ).toBe('Added 1 mod · Removed 3 mods · Updated 1 mod')
  expect(historySummary({ change: HistoryChange.ChangeAdded, name: 'Seed Alpha' })).toBe(
    'Added Seed Alpha',
  )
})

test('historyKind picks an icon class and falls back to other', () => {
  expect(historyKind({ change: HistoryChange.ChangeDisabled })).toBe('disabled')
  expect(historyKind({ change: HistoryChange.ChangeImported })).toBe('added')
  expect(historyKind({ change: HistoryChange.ChangeNote })).toBe('other')
})

test('sentence capitalises the first letter only', () => {
  expect(sentence('added Seed alpha')).toBe('Added Seed alpha')
  expect(sentence('')).toBe('')
})
