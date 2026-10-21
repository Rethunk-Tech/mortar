import { expect, test } from 'bun:test'
import { shareInFlight } from './listProfiles.ts'

test('reads asked for at the same time share one call, and a later read asks again', async () => {
  const share = shareInFlight<number>()
  let calls = 0
  const read = () => {
    calls += 1
    return Promise.resolve(calls)
  }
  const [a, b] = await Promise.all([share('stardew', read), share('stardew', read)])
  expect([a, b, calls]).toEqual([1, 1, 1])
  expect(await share('stardew', read)).toBe(2)
  expect(await share('valheim', read)).toBe(3)
})
