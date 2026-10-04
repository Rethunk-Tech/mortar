import { expect, test } from 'bun:test'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { copyName, duplicatePreset, playPresets, presetNameError } from './profilePresets.ts'

const profile = (extra: Partial<Profile>) => ({ id: 'a', name: 'Farm', ...extra }) as Profile

test('a profile without named presets has no Play presets', () => {
  expect(playPresets(profile({}))).toEqual([])
  expect(playPresets(undefined)).toEqual([])
})

test('the base preset is the default until a named one is', () => {
  const launchPresets = [{ id: 'p1', name: 'Debug' }]
  const base = playPresets(profile({ launchPresets }))
  expect(base.map((p) => [p.name, p.isDefault])).toEqual([
    ['Standard', true],
    ['Debug', false],
  ])
  const named = playPresets(profile({ launchPresets, defaultLaunchPreset: 'p1' }))
  expect(named.map((p) => p.isDefault)).toEqual([false, true])
  expect(named[1]?.key).toBe('p1')
})

test('a dangling default falls back to the base preset', () => {
  const got = playPresets(
    profile({ launchPresets: [{ id: 'p1', name: 'D' }], defaultLaunchPreset: 'x' }),
  )
  expect(got[0]?.isDefault).toBe(true)
})

test('copy names stay unique', () => {
  expect(copyName('Debug', ['Debug'])).toBe('Debug copy')
  expect(copyName('Debug', ['Debug', 'debug copy'])).toBe('Debug copy 2')
})

test('duplicating drops the id so the backend assigns one', () => {
  const src = { id: 'p1', name: 'Debug', launchOptions: '--x' }
  expect(duplicatePreset(src, [src])).toEqual({ id: '', name: 'Debug copy', launchOptions: '--x' })
})

test('name validation mirrors the backend', () => {
  const list = [{ id: 'p1', name: 'Debug' }]
  expect(presetNameError(' ', list, '')).toBe('empty')
  expect(presetNameError('standard', list, '')).toBe('taken')
  expect(presetNameError('DEBUG', list, '')).toBe('taken')
  expect(presetNameError('Debug', list, 'p1')).toBeNull()
  expect(presetNameError('New', list, '')).toBeNull()
})
