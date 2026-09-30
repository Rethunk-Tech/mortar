import { afterEach, beforeEach, expect, jest, test } from 'bun:test'
import { useToasts } from './store.ts'

beforeEach(() => {
  jest.useFakeTimers()
})

afterEach(() => {
  jest.useRealTimers()
  useToasts.setState({ toasts: [] })
})

test('info and success dismiss after 5s; warning and error stay', () => {
  const { push } = useToasts.getState()
  push({ kind: 'info', title: 'a' })
  push({ kind: 'success', title: 'b' })
  push({ kind: 'warning', title: 'c' })
  push({ kind: 'error', title: 'd' })
  jest.advanceTimersByTime(4999)
  expect(useToasts.getState().toasts).toHaveLength(4)
  jest.advanceTimersByTime(1)
  expect(useToasts.getState().toasts.map((t) => t.title)).toEqual(['c', 'd'])
})

test('toasts keep push order and dismiss removes one by id', () => {
  const { push, dismiss } = useToasts.getState()
  const first = push({ kind: 'error', title: 'first' })
  push({ kind: 'error', title: 'second' })
  expect(useToasts.getState().toasts.map((t) => t.title)).toEqual(['first', 'second'])
  dismiss(first)
  expect(useToasts.getState().toasts.map((t) => t.title)).toEqual(['second'])
})
