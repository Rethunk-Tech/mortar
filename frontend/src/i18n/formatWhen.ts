import { i18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { relativeWhen, type WhenOptions } from './when.ts'

export const formatWhen = (value: string | number | Date, options: WhenOptions = {}) =>
  relativeWhen(value, {
    locale: i18n.locale,
    fewSeconds: i18n._(msg`a few seconds ago`),
    ...options,
  })
