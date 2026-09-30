import { beforeEach, expect, mock, test } from 'bun:test'

let installs = 0

mock.module('../../bindings/github.com/Rethunk-AI/mortar/internal/updatesvc/service.ts', () => ({
  Check: async () => null,
  Info: async () => ({ off: false, version: '', channel: '' }),
  Install: async () => {
    installs += 1
    await Promise.resolve()
  },
  Restart: async () => undefined,
}))

const { getInitialState, useMortarUpdate } = await import('./updates.ts')

beforeEach(() => {
  installs = 0
  useMortarUpdate.setState(getInitialState(), true)
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
