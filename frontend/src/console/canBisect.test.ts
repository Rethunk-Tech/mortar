import { expect, test } from 'bun:test'
import { canBisectCrash } from './canBisect.ts'

test('canBisectCrash is true when no cause was identified', () => {
  expect(canBisectCrash({ cause: null })).toBe(true)
  expect(canBisectCrash({ cause: undefined })).toBe(true)
  expect(canBisectCrash({ cause: { modKey: 'a' } })).toBe(false)
})
