import { expect, test } from 'bun:test'
import { i18n } from '@lingui/core'

i18n.load('en', {})
i18n.activate('en')

const { playStyleName, saveKind, saveName } = await import('./saveName.ts')

test('a save is named by its farm, its Lethal Company slot, or its folder or file', () => {
  expect(saveName({ farm: 'Sunny', folder: 'Farm_1' })).toBe('Sunny')
  expect(saveName({ farm: '', folder: 'Farm_1' })).toBe('Farm_1')
  expect(saveName({ farm: '', folder: 'LCSaveFile2' })).toBe('Save file 2')
  expect(saveName({ farm: '', folder: 'LCChallengeFile' })).toBe('Challenge moon')
  expect(saveName({ farm: '', folder: 'Ragnar.fch' })).toBe('Ragnar')
  expect(saveName({ farm: '', folder: 'worlds_local/Midgard.fwl' })).toBe('Midgard')
  expect(saveName({ farm: '', folder: 'Bjornheim-1.vcdbs', name: 'Bjornheim' })).toBe('Bjornheim')
})

test("a Vintage Story world's play style is named, and a new code shows as itself", () => {
  expect(playStyleName('wildernesssurvival')).toBe('Wilderness Survival')
  expect(playStyleName('newstyle')).toBe('newstyle')
  expect(playStyleName(undefined)).toBe('')
})

test('a Valheim save is a character or a world by its folder', () => {
  expect(saveKind('characters_local/Ragnar.fch')).toBe('Character')
  expect(saveKind('worlds_local/Midgard.fwl')).toBe('World')
  expect(saveKind('Farm_1')).toBe('')
})

test('a Lethal Company save says whether it is a slot, an extra slot from a mod or the weekly challenge', () => {
  expect(saveKind('LCSaveFile3')).toBe('Save slot')
  expect(saveKind('LCSaveFile4')).toBe('Extra slot added by a mod')
  expect(saveKind('LCChallengeFile')).toBe('Weekly challenge')
})
