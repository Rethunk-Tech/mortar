import { afterEach, beforeEach, expect, jest, test } from 'bun:test'
import { useToasts } from './store.ts'

beforeEach(() => {
  jest.useFakeTimers()
})

afterEach(() => {
  jest.useRealTimers()
  useToasts.setState(useToasts.getInitialState(), true)
})

test('info and success dismiss after 5s; warning and error after 10s', () => {
  const { push } = useToasts.getState()
  push({ kind: 'info', title: 'a' })
  push({ kind: 'success', title: 'b' })
  push({ kind: 'warning', title: 'c' })
  push({ kind: 'error', title: 'd' })
  jest.advanceTimersByTime(4999)
  expect(useToasts.getState().toasts).toHaveLength(3)
  jest.advanceTimersByTime(1)
  expect(useToasts.getState().toasts.map((t) => t.title)).toEqual(['c', 'd'])
  jest.advanceTimersByTime(5000)
  expect(useToasts.getState().toasts).toHaveLength(0)
})

test('only the newest three show', () => {
  const { push } = useToasts.getState()
  for (const title of ['a', 'b', 'c', 'd']) {
    push({ kind: 'error', title })
  }
  expect(useToasts.getState().toasts.map((t) => t.title)).toEqual(['b', 'c', 'd'])
})

test('hover holds a toast and leaving restarts its countdown', () => {
  const { push, hold, release } = useToasts.getState()
  const id = push({ kind: 'error', title: 'a' })
  hold(id)
  jest.advanceTimersByTime(60_000)
  expect(useToasts.getState().toasts).toHaveLength(1)
  release(id)
  jest.advanceTimersByTime(9999)
  expect(useToasts.getState().toasts).toHaveLength(1)
  jest.advanceTimersByTime(1)
  expect(useToasts.getState().toasts).toHaveLength(0)
})

test('toasts keep push order and dismiss removes one by id', () => {
  const { push, dismiss } = useToasts.getState()
  const first = push({ kind: 'error', title: 'first' })
  push({ kind: 'error', title: 'second' })
  expect(useToasts.getState().toasts.map((t) => t.title)).toEqual(['first', 'second'])
  dismiss(first)
  expect(useToasts.getState().toasts.map((t) => t.title)).toEqual(['second'])
})

test('a toast keeps its picture and action', () => {
  const run = () => undefined
  useToasts.getState().push({
    kind: 'success',
    title: 'SpaceCore installed',
    picture: 'https://example.test/mod.png',
    action: { label: 'Undo', run, profileId: 'p1' },
  })
  const [toast] = useToasts.getState().toasts
  expect(toast?.picture).toBe('https://example.test/mod.png')
  expect(toast?.action?.label).toBe('Undo')
  expect(toast?.action?.run).toBe(run)
  expect(toast?.action?.profileId).toBe('p1')
})

test('history keeps the last 200 toasts newest first after they leave the stack', () => {
  const { push } = useToasts.getState()
  for (let i = 0; i < 201; i += 1) {
    push({ kind: 'info', title: `n${i}`, body: `b${i}` })
  }
  jest.advanceTimersByTime(5000)
  const { history, unread, toasts } = useToasts.getState()
  expect(toasts).toHaveLength(0)
  expect(history).toHaveLength(200)
  expect(history[0]?.title).toBe('n200')
  expect(history[0]?.body).toBe('b200')
  expect(history[0]?.kind).toBe('info')
  expect(typeof history[0]?.at).toBe('number')
  expect(history.at(-1)?.title).toBe('n1')
  expect(unread).toBe(200)
})

test('history keeps actions and Clear drops the list', () => {
  const run = () => undefined
  let latest = true
  useToasts.getState().push({
    kind: 'success',
    title: 'SpaceCore installed',
    action: {
      label: 'Undo',
      run,
      profileId: 'p1',
      live: () =>
        latest
          ? { disabled: false }
          : { disabled: true, reason: 'This is no longer the latest change to that mod.' },
    },
  })
  const [item] = useToasts.getState().history
  expect(item?.action?.run).toBe(run)
  expect(item?.action?.live?.()).toEqual({ disabled: false })
  latest = false
  expect(item?.action?.live?.().disabled).toBe(true)
  useToasts.getState().markRead()
  expect(useToasts.getState().unread).toBe(0)
  useToasts.getState().clearHistory()
  expect(useToasts.getState().history).toEqual([])
  expect(useToasts.getState().unread).toBe(0)
})

test('merges repeated titles and keeps detail and picture in history', () => {
  const { push } = useToasts.getState()
  push({ kind: 'success', title: 'Imported Farm', detail: 'details', picture: 'farm.png' })
  jest.advanceTimersByTime(1000)
  push({ kind: 'success', title: 'Imported Farm', detail: 'new details', picture: 'new.png' })
  const { history } = useToasts.getState()
  expect(history).toHaveLength(1)
  expect(history[0]?.count).toBe(2)
  expect(history[0]?.detail).toBe('details')
  expect(history[0]?.picture).toBe('farm.png')
})
