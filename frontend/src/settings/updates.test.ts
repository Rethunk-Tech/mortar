import { afterEach, beforeEach, expect, mock, spyOn, test } from 'bun:test'
import type { Info } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/updatesvc/models.ts'
import * as updatesvc from '../../bindings/github.com/Rethunk-Tech/mortar/internal/updatesvc/service.ts'
import * as quitservice from '../../bindings/github.com/Rethunk-Tech/mortar/quitservice.ts'
import * as quit from '../quit.ts'
import { stubBinding } from '../testing/stubBinding.ts'
import { useMortarUpdate } from './updates.ts'

let installs = 0

beforeEach(() => {
  installs = 0
  stubBinding(updatesvc, 'Check', () => null)
  stubBinding(updatesvc, 'Info', () => ({ version: '1.0.0', commit: '', off: '' }))
  stubBinding(updatesvc, 'Install', async () => {
    installs += 1
    await Promise.resolve()
  })
  stubBinding(updatesvc, 'Restart', () => undefined)
  stubBinding(quitservice, 'BusySummary', () => '')
  spyOn(quit, 'askQuit').mockResolvedValue(true)
  useMortarUpdate.setState(useMortarUpdate.getInitialState(), true)
})
afterEach(() => {
  mock.restore()
})

test('a check finds Mortar current, and checking again keeps a found or staged update on offer', async () => {
  await useMortarUpdate.getState().check()
  expect(useMortarUpdate.getState().phase).toBe('current')
  for (const phase of ['available', 'ready'] as const) {
    useMortarUpdate.setState(
      {
        ...useMortarUpdate.getInitialState(),
        info: { version: '1.0.0', commit: '', off: '' } satisfies Info,
        phase,
      },
      true,
    )
    await useMortarUpdate.getState().check()
    expect(useMortarUpdate.getState().phase).toBe(phase)
  }
})

test('install runs once from available and ignores a second click', async () => {
  useMortarUpdate.setState({ phase: 'available' })
  const first = useMortarUpdate.getState().install()
  const second = useMortarUpdate.getState().install()
  expect(useMortarUpdate.getState().phase).toBe('installing')
  await Promise.all([first, second])
  expect(installs).toBe(1)
  expect(useMortarUpdate.getState().phase).toBe('ready')
})

test('install is a no-op unless the phase is available', async () => {
  await useMortarUpdate.getState().install()
  expect(installs).toBe(0)
  expect(useMortarUpdate.getState().phase).toBe('idle')
})
