import { expect, test } from 'bun:test'
import { useProfiles } from '../profiles/store.ts'

let started = 0
const realSetInterval = globalThis.setInterval
globalThis.setInterval = ((fn: () => void, ms?: number) => {
  started += 1
  return realSetInterval(fn, ms)
}) as typeof setInterval
await import('./updates.ts')

test('mod changes on the open profile do not restart the update recheck timer', () => {
  useProfiles.setState({ game: { id: 'stardew' } as never, openId: 'p1' })
  const afterOpen = started
  for (let i = 0; i < 5; i += 1) {
    useProfiles.setState({ profiles: [] })
  }
  expect(started).toBe(afterOpen)
  useProfiles.setState({ openId: 'p2' })
  expect(started).toBe(afterOpen + 1)
  useProfiles.setState({ game: null, openId: '' })
  globalThis.setInterval = realSetInterval
})
