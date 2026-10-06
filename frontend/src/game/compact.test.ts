import { expect, test } from 'bun:test'
import { compactMeta, saveFits } from './compact.ts'

test('compactMeta omits zero update and problem parts', () => {
  expect(compactMeta(42, 3, 1)).toEqual([
    { n: 42, kind: 'mods' },
    { n: 3, kind: 'updates' },
    { n: 1, kind: 'problems' },
  ])
  expect(compactMeta(42, 0, 0)).toEqual([{ n: 42, kind: 'mods' }])
  expect(compactMeta(0, 0, 1)).toEqual([{ n: 1, kind: 'problems' }])
})

test('saveFits treats an empty scan as 0 of 0', () => {
  expect(saveFits([])).toEqual({ fitting: 0, recorded: 0, total: 0 })
  expect(saveFits([{ missing: [] }, { missing: ['x'] }])).toEqual({
    fitting: 1,
    recorded: 2,
    total: 2,
  })
  expect(saveFits([{ unrecorded: true }, { unrecorded: true }])).toEqual({
    fitting: 0,
    recorded: 0,
    total: 2,
  })
})
