import { expect, test } from 'bun:test'
import { readdirSync } from 'node:fs'
import { join } from 'node:path'
import {
  availableLocales,
  bestLocale,
  localeLabel,
  pickerLocales,
  systemLocales,
} from './locales.ts'

test('the frontend and Go ship the same locales', () => {
  const dir = join(import.meta.dir, '../../../internal/i18n/locales')
  const shipped: string[] = [...availableLocales]
  const byCode = (a: string, b: string) => a.localeCompare(b)
  expect(shipped.sort(byCode)).toEqual(readdirSync(dir).sort(byCode))
})

const all = [...availableLocales]

test('the closest locale is the same code, then the longest prefix, then the same base language', () => {
  expect(bestLocale('es-AR', all)).toBe('es-AR')
  expect(bestLocale('es-pa', all)).toBe('es-PA')
  expect(bestLocale('es-MX', all)).toBe('es')
  expect(bestLocale('es', all)).toBe('es')
  expect(bestLocale('pt-PT', all)).toBe('pt-BR')
  expect(bestLocale('pt', all)).toBe('pt-BR')
  expect(bestLocale('en-GB', all)).toBe('en')
  expect(bestLocale('fr-FR', all)).toBe('en')
  expect(bestLocale('', all)).toBe('en')
})

test('a regional locale falls back to its base language when that is shipped', async () => {
  const config = (await import('../../lingui.config.ts')).default
  expect(config.fallbackLocales).toEqual({ 'es-AR': 'es', 'es-PA': 'es' })
})

const shares: Record<string, number> = { es: 0.85, 'es-AR': 0.9, 'es-PA': 0.1, 'pt-BR': 0 }
const complete = (locale: string) => shares[locale] ?? 1

test('the system language skips a locale below 80% and falls through to the next candidate', () => {
  expect(systemLocales(complete)).toEqual(['en', 'es', 'es-AR'])
  const usable = systemLocales(complete)
  expect(bestLocale('es-AR', usable)).toBe('es-AR')
  expect(bestLocale('es-PA', usable)).toBe('es')
  expect(bestLocale('pt-BR', usable)).toBe('en')
})

test('the picker lists English, locales with some translation and the chosen one, with their share', () => {
  expect(pickerLocales('', complete)).toEqual(['en', 'es', 'es-AR', 'es-PA'])
  expect(pickerLocales('pt-BR', complete)).toEqual(['en', 'es', 'es-AR', 'es-PA', 'pt-BR'])
  expect(localeLabel('es-PA', complete)).toMatch(/\(10%\)$/)
  expect(localeLabel('en', complete)).toBe('English')
})
