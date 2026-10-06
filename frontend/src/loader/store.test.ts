import { beforeEach, expect, test } from 'bun:test'
import { useToasts } from '../toasts/store.ts'
import { useLoader } from './store.ts'

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
    perProfile: true,
  }
  useLoader.setState({ status, game: 'stardew' })
  useLoader.getState().check('stardew')
  expect(useLoader.getState().status).toBe(status)
  useLoader.getState().check('lethal-company')
  expect(useLoader.getState().status).toBeNull()
})
