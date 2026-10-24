import { afterEach, beforeEach, expect, mock, spyOn, test } from 'bun:test'
import { CancellablePromise } from '@wailsio/runtime'
import { Hint } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import { State } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import * as launchsvc from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import * as problems from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/service.ts'
import { useLaunch } from './store.ts'

let warnCalls = 0
let startCalls = 0
let askGraphics = false
let graphicsAfterCrash = false
let warnWait: Promise<void> = Promise.resolve()
let finishWarn: (() => void) | undefined

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
  spyOn(problems, 'Problems').mockImplementation(() =>
    CancellablePromise.resolve({
      missing: null,
      duplicates: null,
      broken: null,
      assetConflicts: null,
      settings: null,
      runErrors: null,
      dismissed: [],
      unknown: false,
    }),
  )
  spyOn(problems, 'UpdateWarning').mockImplementation(() => {
    warnCalls += 1
    return CancellablePromise.resolve(
      warnWait.then(() => ({
        changed: true,
        recorded: '1.6.14',
        installed: '1.6.15',
        broken: null,
      })),
    )
  })
  spyOn(launchsvc, 'Status').mockImplementation(() =>
    CancellablePromise.resolve({
      game: 'stardew',
      state: State.Idle,
      profile: '',
      since: 0,
      hint: Hint.$zero,
      error: '',
    }),
  )
  spyOn(launchsvc, 'GraphicsAsk').mockImplementation(() =>
    CancellablePromise.resolve({ ask: askGraphics, afterEarlyExit: graphicsAfterCrash }),
  )
  spyOn(launchsvc, 'Start').mockImplementation(() => {
    startCalls += 1
    return CancellablePromise.resolve()
  })
})
afterEach(() => {
  mock.restore()
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
