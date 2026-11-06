import { describe, expect, test } from 'bun:test'
import { keepUnchanged } from '@rethunk/react-load-hooks'
import { forgetLast, getLast, peekLast, putLast, subscribeLast } from './lastGood.ts'

const countNotifications = (run: () => void) => {
  let calls = 0
  const off = subscribeLast(() => {
    calls += 1
  })
  run()
  off()
  return calls
}

describe('lastGood', () => {
  test('an equal answer keeps the stored object and notifies nobody', () => {
    const first = putLast('eq', { n: 1 })
    let again: { n: number } | undefined
    const calls = countNotifications(() => {
      again = putLast('eq', { n: 1 })
    })
    expect(again).toBe(first)
    expect(calls).toBe(0)
  })

  test('a different answer replaces it and notifies', () => {
    putLast('diff', [1])
    const calls = countNotifications(() => {
      putLast('diff', [1, 2])
    })
    expect(getLast('diff')).toEqual([1, 2])
    expect(calls).toBe(1)
  })

  test('keys are isolated and forgetLast drops by prefix', () => {
    putLast('p:a', 1)
    putLast('p:b', 2)
    putLast('q:a', 3)
    forgetLast('p:')
    expect(getLast('p:a')).toBeUndefined()
    expect(getLast('p:b')).toBeUndefined()
    expect(getLast('q:a')).toBe(3)
  })

  test('a seed never replaces a live answer', () => {
    putLast('live', 'live')
    expect(putLast('live', 'old', { seed: true })).toBe('live')
    expect(peekLast('live')?.seed).toBe(false)
  })

  test('a live answer equal to a seed keeps the object and clears the seed mark', () => {
    const seeded = putLast('seeded', { a: 1 }, { seed: true })
    expect(peekLast('seeded')?.seed).toBe(true)
    const calls = countNotifications(() => {
      expect(putLast('seeded', { a: 1 })).toBe(seeded)
    })
    expect(peekLast('seeded')?.seed).toBe(false)
    expect(calls).toBe(1)
  })

  test('keep merges a list so unchanged rows keep identity', () => {
    interface Row {
      id: string
      v: number
    }
    const rows: Row[] = [
      { id: 'a', v: 1 },
      { id: 'b', v: 2 },
    ]
    putLast('rows', rows)
    const merged = putLast(
      'rows',
      [
        { id: 'a', v: 1 },
        { id: 'b', v: 3 },
      ],
      { keep: (prev: Row[], next: Row[]) => keepUnchanged(prev, next, (r) => r.id) },
    )
    expect(merged[0]).toBe(rows[0] as Row)
    expect(merged[1]?.v).toBe(3)
  })
})
