import { expect, test } from 'bun:test'
import { coalescer } from './coalesce.ts'

test('asks made while a run is under way share one follow-up run', async () => {
  const coalesce = coalescer()
  let started = 0
  let release: () => void = () => undefined
  const run = () => {
    started += 1
    return new Promise<void>((resolve) => {
      release = resolve
    })
  }
  const first = coalesce('p', run)
  const second = coalesce('p', run)
  const third = coalesce('p', run)
  expect(started).toBe(1)
  expect(second).toBe(third)
  release()
  await first
  await new Promise((resolve) => setTimeout(resolve, 0))
  expect(started).toBe(2)
  release()
  await Promise.all([second, third])
  expect(started).toBe(2)
  await coalesce('p', () => Promise.resolve())
})

test('other keys run alongside, and a failed run still lets the follow-up start', async () => {
  const coalesce = coalescer()
  let other = 0
  const failing = coalesce('a', () => Promise.reject(new Error('x')))
  const after = coalesce('a', () => Promise.resolve())
  await coalesce('b', () => {
    other += 1
    return Promise.resolve()
  })
  await expect(failing).rejects.toThrow('x')
  await after
  expect(other).toBe(1)
})
