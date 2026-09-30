import { i18n as global } from '@lingui/core'
import { messages } from '../locales/en/messages.ts'

global.load('en', messages)
global.activate('en')

export const i18n = global
