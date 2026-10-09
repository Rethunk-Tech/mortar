import { beforeEach, expect, mock, test } from 'bun:test'

let warnCalls = 0
let startCalls = 0
let askGraphics = false
let graphicsAfterCrash = false
let warnWait: Promise<void> = Promise.resolve()
let finishWarn: (() => void) | undefined

mock.module('../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/service.ts', () => ({
  Problems: async () => ({
    missing: null,
    duplicates: null,
    broken: null,
    assetConflicts: null,
    settings: null,
    runErrors: null,
    dismissed: [],
    unknown: false,
  }),
  Updates: async () => ({ updates: null, unknown: false }),
  UpdateWarning: () => {
    warnCalls += 1
    return warnWait.then(() => ({
      changed: true,
      recorded: '1.6.14',
      installed: '1.6.15',
      broken: null,
    }))
  },
}))

mock.module('../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts', () => ({
  Status: async () => ({
    game: 'stardew',
    state: 0,
    profile: '',
    since: 0,
    hint: 0,
    error: '',
  }),
  GraphicsAsk: async () => ({ ask: askGraphics, afterEarlyExit: graphicsAfterCrash }),
  Start: async () => {
    startCalls += 1
  },
  StartVanilla: async () => undefined,
  Stop: async () => undefined,
}))

const { useLaunch } = await import('./store.ts')

beforeEach(() => {
  warnCalls = 0
  startCalls = 0
  askGraphics = false
  graphicsAfterCrash = false
  finishWarn = undefined
  warnWait = new Promise<void>((resolve) => {
    finishWarn = resolve
  })
  useLaunch.setState(useLaunch.getInitialState(), true)
})

test('a second Start waits until the first UpdateWarning finishes', async () => {
  const first = useLaunch.getState().start('stardew', 'p1', false)
  expect(useLaunch.getState().starting).toBe(true)
  const second = useLaunch.getState().start('stardew', 'p1', false)
  finishWarn?.()
  await Promise.all([first, second])
  expect(warnCalls).toBe(1)
  expect(startCalls).toBe(0)
  expect(useLaunch.getState().starting).toBe(false)
  expect(useLaunch.getState().updateWarn?.profile).toBe('p1')
})

test('an unanswered graphics choice stops Play at the dialog before any other check', async () => {
  askGraphics = true
  await useLaunch.getState().start('peak', 'p1', false)
  const state = useLaunch.getState()
  expect(state.graphicsAsk).toEqual({
    game: 'peak',
    profile: 'p1',
    direct: false,
    afterEarlyExit: false,
  })
  expect(state.starting).toBe(false)
  expect(warnCalls).toBe(0)
  expect(startCalls).toBe(0)
  state.dismissGraphicsAsk()
  expect(useLaunch.getState().graphicsAsk).toBeNull()
})

test('the dialog after an early exit says so', async () => {
  askGraphics = true
  graphicsAfterCrash = true
  await useLaunch.getState().start('peak', 'p1', false)
  expect(useLaunch.getState().graphicsAsk?.afterEarlyExit).toBe(true)
})
