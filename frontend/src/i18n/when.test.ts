import { expect, test } from 'bun:test'
import { absoluteWhen, relativeWhen } from './when.ts'

const now = Date.parse('2026-10-02T12:00:00Z')
const at = (ms: number) => now - ms
const opts = { now, locale: 'en' }

test('relativeWhen is relative within a week and a date before that', () => {
  expect(relativeWhen(at(5000), opts)).toBe('a few seconds ago')
  expect(relativeWhen(at(-5000), opts)).toBe('a few seconds ago')
  expect(relativeWhen(at(3 * 60_000), opts)).toBe('3 minutes ago')
  expect(relativeWhen(at(2 * 3_600_000 + 1), opts)).toBe('2 hours ago')
  expect(relativeWhen(at(86_400_000), opts)).toBe('yesterday')
  expect(relativeWhen(at(3 * 86_400_000), opts)).toBe('3 days ago')
  expect(relativeWhen('2026-03-15T02:54:41Z', opts)).toBe('Mar 15, 2026')
  expect(relativeWhen('0001-01-01T00:00:00Z', opts)).toBe('')
  expect(relativeWhen('nonsense', opts)).toBe('')
})

test('absoluteWhen uses a medium date and short time and skips a zero date', () => {
  expect(absoluteWhen('0001-01-01T00:00:00Z', 'en')).toBe('')
  expect(absoluteWhen('nonsense', 'en')).toBe('')
  expect(absoluteWhen('2026-03-15T02:54:41Z', 'en')).toBe(
    new Intl.DateTimeFormat('en', { dateStyle: 'medium', timeStyle: 'short' }).format(
      new Date('2026-03-15T02:54:41Z'),
    ),
  )
})
