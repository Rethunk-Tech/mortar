import { expect, test } from 'bun:test'
import { launchLine, launchOptionsSet, shouldShowFirstRun } from './logic.ts'

test('first run shows only for a missing game or a first launch', () => {
  const ready = { installed: true, smapiInstalled: true, profileCount: 1 }
  expect(shouldShowFirstRun(ready)).toBe(false)
  expect(shouldShowFirstRun({ ...ready, installed: false })).toBe(true)
  expect(shouldShowFirstRun({ ...ready, smapiInstalled: false })).toBe(false)
  expect(shouldShowFirstRun({ ...ready, profileCount: 0 })).toBe(false)
  expect(shouldShowFirstRun({ ...ready, smapiInstalled: false, profileCount: 0 })).toBe(true)
})

test('launch options line points at SMAPI and is recognised once pasted', () => {
  const line = launchLine('C:\\Steam\\Stardew Valley')
  expect(line).toBe('"C:\\Steam\\Stardew Valley\\StardewModdingAPI.exe" %command%')
  expect(launchOptionsSet(line)).toBe(true)
  expect(launchOptionsSet('"c:\\x\\stardewmoddingapi.EXE" %command%')).toBe(true)
  expect(launchOptionsSet('"C:\\x\\StardewModdingAPI.exe"')).toBe(false)
  expect(launchOptionsSet('')).toBe(false)
  expect(launchOptionsSet('-novid %command%')).toBe(false)
})
