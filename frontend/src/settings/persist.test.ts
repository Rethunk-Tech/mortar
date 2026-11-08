import { afterEach, beforeEach, expect, test } from 'bun:test'
import { useToasts } from '../toasts/store.ts'
import { pickSetting, saveSetting, settingSaveFailed } from './persist.ts'

const last = () => useToasts.getState().toasts.at(-1)
const settled = () => new Promise((done) => setTimeout(done, 0))

beforeEach(() => useToasts.setState({ toasts: [], history: [] }))
afterEach(() => useToasts.setState(useToasts.getInitialState(), true))

test('every failed setting write is titled as a save and keeps its cause', async () => {
  settingSaveFailed(new Error('open settings.json: read-only file system'))
  expect(last()?.title).toBe('Could not save that setting')
  expect(last()?.detail).toBe('open settings.json: read-only file system')

  useToasts.setState({ toasts: [], history: [] })
  saveSetting(() => Promise.reject(new Error('open settings.json: read-only file system')))
  await settled()
  expect(last()?.title).toBe('Could not save that setting')
  expect(last()?.detail).toBe('open settings.json: read-only file system')
})

test('cancelling a pick saves nothing and reports nothing', async () => {
  pickSetting(() => Promise.resolve(''), 'downloadFolder', '')
  await settled()
  expect(useToasts.getState().toasts).toEqual([])
})
