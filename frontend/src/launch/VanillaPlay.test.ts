import { expect, test } from 'bun:test'
import { windowsVanillaAfterForcesCheck } from './playOpen.ts'

test('a failed ForcesLoader check does not start vanilla', () => {
  expect(windowsVanillaAfterForcesCheck(false, false)).toBe('abort')
  expect(windowsVanillaAfterForcesCheck(false, true)).toBe('abort')
})

test('Windows vanilla play warns when Steam still starts SMAPI', () => {
  expect(windowsVanillaAfterForcesCheck(true, true)).toBe('warn')
  expect(windowsVanillaAfterForcesCheck(true, false)).toBe('start')
})
