import { afterEach, expect, test } from 'bun:test'
import { useIncomingShares } from './incoming.ts'

const arrival = (id: number) => ({
  id,
  sender: 'Alex',
  game: 'stardew',
  payload: '',
  profileName: 'Farm',
  paired: true,
})

afterEach(() => {
  useIncomingShares.setState(useIncomingShares.getInitialState(), true)
})

test('removing an expired share reports whether it was still waiting', () => {
  useIncomingShares.getState().add(arrival(1))
  useIncomingShares.getState().add(arrival(2))
  expect(useIncomingShares.getState().remove(1)).toBe(true)
  expect(useIncomingShares.getState().items.map((a) => a.id)).toEqual([2])
  expect(useIncomingShares.getState().remove(1)).toBe(false)
})
