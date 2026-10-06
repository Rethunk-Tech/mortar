import { describe, expect, mock, test } from 'bun:test'
import type { Status } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/loader/models.ts'
import { loaderToastShownToday, loaderUpdateText } from './loaderToast.ts'

mock.module('@lingui/core/macro', () => ({
  msg: (parts: TemplateStringsArray, ...values: string[]) => String.raw({ raw: parts }, ...values),
  plural: () => '',
}))

const status = (s: Partial<Status>): Status => ({
  installed: true,
  broken: false,
  version: '',
  gameVersion: '',
  latest: '',
  updateAvailable: true,
  perProfile: false,
  ...s,
})

const dayMs = 24 * 60 * 60 * 1000

describe('loaderToastShownToday', () => {
  test('is false when nothing has been stored', () => {
    expect(loaderToastShownToday(undefined)).toBe(false)
    expect(loaderToastShownToday('')).toBe(false)
  })

  test('is true within a day of the stored time', () => {
    const now = Date.parse('2026-09-30T12:00:00.000Z')
    expect(loaderToastShownToday('2026-09-30T00:00:00.000Z', now)).toBe(true)
    expect(loaderToastShownToday('2026-09-29T12:00:01.000Z', now)).toBe(true)
  })

  test('is false after a day or when the stored value is not a date', () => {
    const now = Date.parse('2026-09-30T12:00:00.000Z')
    expect(loaderToastShownToday('2026-09-29T12:00:00.000Z', now)).toBe(false)
    expect(loaderToastShownToday('nope', now)).toBe(false)
    expect(now - Date.parse('2026-09-29T12:00:00.000Z')).toBe(dayMs)
  })
})

describe('loaderUpdateText', () => {
  test('names the game loader, not SMAPI', () => {
    expect(loaderUpdateText('SMAPI', status({ latest: '4.3.2', version: '4.3.1' }))).toEqual({
      title: 'SMAPI 4.3.2 is out',
    })
  })

  test('a loader in each profile says Update replaces every profile copy', () => {
    const text = loaderUpdateText(
      'BepInEx',
      status({ latest: '5.4.23.3', version: '5.4.23.2', perProfile: true }),
    )
    expect(text.title).toBe('BepInEx 5.4.23.3 is out')
    expect(text.body).toBe("Your profiles have 5.4.23.2. Update replaces each profile's copy.")
  })
})
