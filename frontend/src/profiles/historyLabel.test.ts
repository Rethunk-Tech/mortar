import { expect, mock, test } from 'bun:test'
import { HistoryChange } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

mock.module('@lingui/core/macro', () => ({
  // A named placeholder arrives as { name: value }.
  msg: (parts: TemplateStringsArray, ...values: unknown[]) =>
    String.raw(
      { raw: parts },
      ...values.map((v) => (v !== null && typeof v === 'object' ? Object.values(v)[0] : v)),
    ),
  plural: (n: number, forms: { one: string; other: string }) =>
    (n === 1 ? forms.one : forms.other).replace('#', String(n)),
}))

const { historyLabel } = await import('./historyLabel.ts')

test('historyLabel words each change from its fields', () => {
  expect(historyLabel({ change: HistoryChange.ChangeDisabled, name: 'Seed Alpha' })).toBe(
    'Disabled Seed Alpha',
  )
  expect(
    historyLabel({ change: HistoryChange.ChangeUpdated, name: 'Beta', from: '1.0', to: '1.1' }),
  ).toBe('Updated Beta from 1.0 to 1.1')
  expect(historyLabel({ change: HistoryChange.ChangeMods, count: 3 })).toBe('Changed 3 mods')
  expect(historyLabel({ change: HistoryChange.ChangeMods, count: 1 })).toBe('Changed mods')
  expect(historyLabel({ change: HistoryChange.ChangeImported, count: 1 })).toBe('Imported 1 mod')
  expect(historyLabel({ change: HistoryChange.$zero })).toBe('Changed the profile')
})

test('a revert names its time through the date formatter, not as RFC 3339', () => {
  const label = historyLabel({
    change: HistoryChange.ChangeReverted,
    target: '2020-10-06T04:48:44Z',
  })
  expect(label.startsWith('Went back to ')).toBe(true)
  expect(label).not.toContain('T04:48:44Z')
  expect(label).toContain('2020')
})
