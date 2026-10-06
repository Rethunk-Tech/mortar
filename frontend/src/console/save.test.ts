import { expect, test } from 'bun:test'
import {
  type Entry,
  Level,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import { logFileName, saveLogText } from './save.ts'

const line = (message: string): Entry => ({
  seq: 1,
  time: '19:43:50',
  level: Level.Info,
  mod: 'SMAPI',
  message,
  cont: false,
})

test('the default save name is the loader, the profile, and the date', () => {
  expect(logFileName('SMAPI', 'Main', new Date(2026, 8, 30))).toBe('SMAPI-Main-2026-09-30.txt')
  expect(logFileName('SMAPI', 'Farm/A', new Date(2026, 0, 2))).toBe('SMAPI-Farm-A-2026-01-02.txt')
  expect(logFileName('BepInEx', 'Main', new Date(2026, 8, 30))).toBe('BepInEx-Main-2026-09-30.txt')
  expect(logFileName('', '', new Date(2026, 8, 30))).toBe('log-profile-2026-09-30.txt')
})

test('the raw SMAPI log is saved when the profile owns it, else the console lines', () => {
  expect(saveLogText('[SMAPI] hello', [line('parsed')])).toBe('[SMAPI] hello')
  expect(saveLogText('', [line('parsed')])).toBe('[19:43:50 INFO  SMAPI] parsed')
})
