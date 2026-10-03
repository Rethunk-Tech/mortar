import { expect, test } from 'bun:test'
import { applyLaunchPreset } from './applyLaunchPreset.ts'

test('applyLaunchPreset fills options, prefix, and env from a preset', () => {
  expect(
    applyLaunchPreset({ options: '--developer-mode', prefix: 'gamemoderun', env: 'MANGOHUD=1' }),
  ).toEqual({
    options: '--developer-mode',
    prefix: 'gamemoderun',
    env: 'MANGOHUD=1',
  })
})

test('applyLaunchPreset treats missing fields as empty', () => {
  expect(applyLaunchPreset({})).toEqual({ options: '', prefix: '', env: '' })
})
