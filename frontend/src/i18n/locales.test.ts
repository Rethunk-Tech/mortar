import { expect, test } from 'bun:test'
import { readdirSync } from 'node:fs'
import { join } from 'node:path'
import { availableLocales, bestLocale } from './locales.ts'

test('the frontend and Go ship the same locales', () => {
  const dir = join(import.meta.dir, '../../../internal/i18n/locales')
  const shipped: string[] = [...availableLocales]
  const byCode = (a: string, b: string) => a.localeCompare(b)
  expect(shipped.sort(byCode)).toEqual(readdirSync(dir).sort(byCode))
})

test('the closest locale is the same code, then the longest prefix, then the same base language', () => {
  expect(bestLocale('es-AR')).toBe('es-AR')
  expect(bestLocale('es-pa')).toBe('es-PA')
  expect(bestLocale('es-MX')).toBe('es')
  expect(bestLocale('es')).toBe('es')
  expect(bestLocale('pt-PT')).toBe('pt-BR')
  expect(bestLocale('pt')).toBe('pt-BR')
  expect(bestLocale('en-GB')).toBe('en')
  expect(bestLocale('fr-FR')).toBe('en')
  expect(bestLocale('')).toBe('en')
})

test('a regional locale falls back to its base language when that is shipped', async () => {
  const config = (await import('../../lingui.config.ts')).default
  expect(config.fallbackLocales).toEqual({ 'es-AR': 'es', 'es-PA': 'es' })
})
