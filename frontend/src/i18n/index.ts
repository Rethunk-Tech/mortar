import { i18n as global } from '@lingui/core'
import { messages } from '../locales/en/messages.ts'
import { type AvailableLocale, availableLocales, isAvailableLocale } from './locales.ts'

global.load('en', messages)
global.activate('en')

function bestLocale(): AvailableLocale {
  const language = navigator.language.toLowerCase()
  return (
    availableLocales.find((locale) => language === locale || language.startsWith(`${locale}-`)) ??
    'en'
  )
}

export async function activateLanguage(setting: string): Promise<void> {
  const locale = isAvailableLocale(setting) ? setting : bestLocale()
  if (locale === 'en') {
    global.activate(locale)
    return
  }
  const catalog = await import(`../locales/${locale}/messages.ts`)
  global.load(locale, catalog.messages)
  global.activate(locale)
}

export const i18n = global
