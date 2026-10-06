import { expect, mock, test } from 'bun:test'

mock.module('@lingui/core/macro', () => ({
  msg: (parts: TemplateStringsArray, ...values: unknown[]) => String.raw({ raw: parts }, ...values),
}))

const { saveName } = await import('./saveName.ts')

test('a save is named by its farm, its Lethal Company slot, or its folder', () => {
  expect(saveName({ farm: 'Sunny', folder: 'Farm_1' })).toBe('Sunny')
  expect(saveName({ farm: '', folder: 'Farm_1' })).toBe('Farm_1')
  expect(saveName({ farm: '', folder: 'LCSaveFile2' })).toBe('Save file 2')
  expect(saveName({ farm: '', folder: 'LCChallengeFile' })).toBe('Challenge moon')
})
