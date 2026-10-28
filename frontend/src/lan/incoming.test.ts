import { afterEach, beforeEach, expect, mock, test } from 'bun:test'
import * as lan from '../../bindings/github.com/Rethunk-Tech/mortar/internal/lan/service.ts'
import { stubBinding } from '../testing/stubBinding.ts'
import { useIncomingShares } from './incoming.ts'

const dismissed: number[] = []
const arrival = (id: number) => ({
  id,
  sender: 'Alex',
  game: 'stardew',
  payload: '',
  profileName: 'Farm',
  paired: true,
})

beforeEach(() => {
  stubBinding(lan, 'Dismiss', (id: number) => {
    dismissed.push(id)
  })
  stubBinding(lan, 'Pending', () => [])
  stubBinding(lan, 'OutgoingTransfers', () => [])
})
afterEach(() => {
  mock.restore()
  useIncomingShares.setState(useIncomingShares.getInitialState(), true)
  dismissed.length = 0
})

test('removing an expired share reports whether it was still waiting', () => {
  useIncomingShares.getState().add(arrival(1))
  useIncomingShares.getState().add(arrival(2))
  expect(useIncomingShares.getState().remove(1)).toBe(true)
  expect(useIncomingShares.getState().items.map((a) => a.id)).toEqual([2])
  expect(useIncomingShares.getState().remove(1)).toBe(false)
  expect(dismissed).toEqual([])
})

test('answering a share drops it from the service so a reload does not bring it back', () => {
  useIncomingShares.getState().add(arrival(1))
  useIncomingShares.getState().add(arrival(2))
  useIncomingShares.getState().removeFirst()
  expect(dismissed).toEqual([1])
  expect(useIncomingShares.getState().items.map((a) => a.id)).toEqual([2])
})

test('an outgoing transfer keeps its latest progress', () => {
  const transfer = {
    id: 1,
    peer: 'Alex',
    profile: 'Farm',
    current: 0,
    total: 2,
    bytes: 0,
    totalBytes: 40,
    state: 'sending',
  }
  useIncomingShares.getState().setOutgoing(transfer)
  useIncomingShares.getState().setOutgoing({ ...transfer, current: 2, state: 'done' })
  expect(useIncomingShares.getState().outgoing[1]?.state).toBe('done')
})
