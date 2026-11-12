import { expect, test } from 'bun:test'
import { createSignal } from './signal.ts'

test('emit reaches every subscriber until it unsubscribes', () => {
  const s = createSignal()
  const calls: string[] = []
  const offA = s.subscribe(() => calls.push('a'))
  s.subscribe(() => calls.push('b'))
  s.emit()
  offA()
  s.emit()
  expect(calls).toEqual(['a', 'b', 'b'])
})
