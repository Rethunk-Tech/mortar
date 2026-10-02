import { beforeEach, expect, mock, test } from 'bun:test'
import type { Info } from '../../bindings/github.com/Rethunk-AI/mortar/internal/updatesvc/models.ts'

let installs = 0

mock.module('../../bindings/github.com/Rethunk-AI/mortar/internal/updatesvc/service.ts', () => ({
  Check: async () => null,
  Info: async () => ({ version: '1.0.0', off: '' }),
  Install: async () => {
    installs += 1
    await Promise.resolve()
  },
  Restart: async () => undefined,
}))
mock.module('../quit.ts', () => ({ askQuit: async () => true }))

const { getInitialState, useMortarUpdate } = await import('./updates.ts')

beforeEach(() => {
  installs = 0
  useMortarUpdate.setState(getInitialState(), true)
})

test('checking again keeps a found or staged update on offer', async () => {
  for (const phase of ['available', 'ready'] as const) {
    useMortarUpdate.setState(
      {
        ...useMortarUpdate.getInitialState(),
        info: { version: '1.0.0', off: '' } satisfies Info,
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
