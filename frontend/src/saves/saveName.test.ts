import { expect, mock, test } from 'bun:test'
import { i18n } from '@lingui/core'

mock.module('@lingui/core/macro', () => ({
  msg: (parts: TemplateStringsArray, ...values: unknown[]) => String.raw({ raw: parts }, ...values),
}))

// The mocked msg hands Lingui the English text as its id, which an empty catalog returns as is.
i18n.load('en', {})
i18n.activate('en')

const { saveName } = await import('./saveName.ts')

test('a save is named by its farm, its Lethal Company slot, or its folder or file', () => {
  expect(saveName({ farm: 'Sunny', folder: 'Farm_1' })).toBe('Sunny')
  expect(saveName({ farm: '', folder: 'Farm_1' })).toBe('Farm_1')
  expect(saveName({ farm: '', folder: 'LCSaveFile2' })).toBe('Save file 2')
  expect(saveName({ farm: '', folder: 'LCChallengeFile' })).toBe('Challenge moon')
  expect(saveName({ farm: '', folder: 'Ragnar.fch' })).toBe('Ragnar')
})
