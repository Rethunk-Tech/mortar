import { expect, test } from 'bun:test'
import { follow } from './follow.ts'

test('a fetch that finishes after an event is dropped', async () => {
  const applied: Array<{ v: number; first: boolean }> = []
  let emit:
    | ((ev: { data: { items: never[]; paused: boolean; limitedUntil: number } }) => void)
    | undefined
  let release: () => void = () => undefined
  const gate = new Promise<void>((resolve) => {
    release = resolve
  })
  const done = follow(
    'queue:changed',
    async () => {
      await gate
      return { items: [], paused: false, limitedUntil: 1 }
    },
    (next, first) => {
      applied.push({ v: next.limitedUntil, first })
    },
    (_name, cb) => {
      emit = cb
    },
  )
  emit?.({ data: { items: [], paused: false, limitedUntil: 2 } })
  release()
  await done
  expect(applied).toEqual([{ v: 2, first: true }])
})

test('the initial fetch applies when no event has arrived', async () => {
  const applied: Array<{ v: number; first: boolean }> = []
  await follow(
    'queue:changed',
    async () => ({ items: [], paused: false, limitedUntil: 3 }),
    (next, first) => {
      applied.push({ v: next.limitedUntil, first })
    },
    () => undefined,
  )
  expect(applied).toEqual([{ v: 3, first: true }])
})
