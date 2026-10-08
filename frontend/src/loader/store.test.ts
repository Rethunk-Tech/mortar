import { beforeEach, expect, mock, test } from 'bun:test'
import * as loadersvc from '../../bindings/github.com/Rethunk-Tech/mortar/internal/loadersvc/service.ts'
import { useNav } from '../nav/store.ts'
import { useToasts } from '../toasts/store.ts'
import { useLoader } from './store.ts'

mock.module('@lingui/core/macro', () => ({
  msg: (parts: TemplateStringsArray) => parts.join(''),
  plural: () => '',
}))

let installFails = false
mock.module('../../bindings/github.com/Rethunk-Tech/mortar/internal/loadersvc/service.ts', () => ({
  ...loadersvc,
  Install: async () => {
    if (installFails) {
      throw new Error('no network')
    }
    return { installed: true, version: '4.5.2' }
  },
}))

beforeEach(() => {
  useLoader.setState(useLoader.getInitialState(), true)
  useToasts.setState(useToasts.getInitialState(), true)
})

test('a second install while one is in flight does nothing', async () => {
  useLoader.setState({ pending: true })
  await useLoader.getState().install('stardew')
  expect(useLoader.getState().pending).toBe(true)
  expect(useLoader.getState().error).toBe('')
  expect(useToasts.getState()).toEqual(useToasts.getInitialState())
})

test("a recheck keeps the same game's status and drops another game's", () => {
  const status = {
    installed: true,
    broken: false,
    version: '4.5.2',
    gameVersion: '1.6.15',
    latest: '4.5.2',
    updateAvailable: false,
    shared: false,
    linkedFrom: '',
    perProfile: true,
  }
  useLoader.setState({ status, game: 'stardew' })
  useLoader.getState().check('stardew')
  expect(useLoader.getState().status).toBe(status)
  useLoader.getState().check('lethal-company')
  expect(useLoader.getState().status).toBeNull()
})

test('an install from setup toasts neither success nor failure; the loader step shows both', async () => {
  useNav.setState({ route: { name: 'game-setup', game: 'stardew' } })
  await useLoader.getState().install('stardew')
  expect(useLoader.getState().status?.installed).toBe(true)
  installFails = true
  await useLoader.getState().install('stardew')
  expect(useLoader.getState().error).not.toBe('')
  expect(useToasts.getState().toasts).toEqual([])
})
