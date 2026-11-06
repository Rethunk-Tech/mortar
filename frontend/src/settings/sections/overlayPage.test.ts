import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { overlayPreview } from './overlayUrl.ts'

// The served overlay page formats the Stardew fields itself; its reader is run here and held to the settings preview's.
const html = readFileSync(
  join(import.meta.dir, '../../../../internal/overlay/overlay.html'),
  'utf8',
)
const start = html.indexOf('const CLOCK_HOUR')
const end = html.indexOf('function renderGame')
const pageReader = new Function(`${html.slice(start, end)}\nreturn valueFor`)() as (
  name: string,
  data: Record<string, unknown>,
) => string

const full = {
  dayOfWeek: 'Tuesday',
  daysLeftInSeason: 9,
  friendName: 'Abigail',
  friendHearts: 8,
  petName: 'Rex',
  petType: 'Dog',
  petWatered: false,
}
const FIELDS = ['weekday', 'daysLeft', 'friend', 'pet', 'petWatered']

test('the page shows the text of each Stardew calendar, friend and pet field', () => {
  expect(FIELDS.map((f) => pageReader(f, full))).toEqual([
    'Tuesday',
    '9',
    'Abigail 8',
    'Rex (Dog)',
    'No',
  ])
  expect(pageReader('petWatered', { petWatered: true })).toBe('Yes')
})

test('the page and the settings preview show the same text, and both leave out what the bridge omits', () => {
  for (const body of [full, { friendName: 'Abigail', petName: 'Rex' }, {}]) {
    for (const f of FIELDS) {
      const preview = overlayPreview({ ok: true, body: { inGame: true, ...body } }, f)
      expect(preview).toEqual({ kind: 'value', text: pageReader(f, body) })
    }
  }
})
