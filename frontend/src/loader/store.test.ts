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
