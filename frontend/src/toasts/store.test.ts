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
