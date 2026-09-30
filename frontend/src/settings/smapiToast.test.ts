import { describe, expect, test } from 'bun:test'
import { smapiToastShownToday } from './smapiToast.ts'

const dayMs = 24 * 60 * 60 * 1000

describe('smapiToastShownToday', () => {
  test('is false when nothing has been stored', () => {
    expect(smapiToastShownToday(undefined)).toBe(false)
    expect(smapiToastShownToday('')).toBe(false)
  })

  test('is true within a day of the stored time', () => {
    const now = Date.parse('2026-09-30T12:00:00.000Z')
    expect(smapiToastShownToday('2026-09-30T00:00:00.000Z', now)).toBe(true)
    expect(smapiToastShownToday('2026-09-29T12:00:01.000Z', now)).toBe(true)
  })

  test('is false after a day or when the stored value is not a date', () => {
    const now = Date.parse('2026-09-30T12:00:00.000Z')
    expect(smapiToastShownToday('2026-09-29T12:00:00.000Z', now)).toBe(false)
    expect(smapiToastShownToday('nope', now)).toBe(false)
    expect(now - Date.parse('2026-09-29T12:00:00.000Z')).toBe(dayMs)
  })
})
