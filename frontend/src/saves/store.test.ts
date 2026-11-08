import { afterEach, expect, mock, test } from 'bun:test'
import * as savessvc from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/service.ts'
import { stubBinding } from '../testing/stubBinding.ts'
import { useSaves } from './store.ts'

afterEach(() => {
  mock.restore()
  useSaves.setState(useSaves.getInitialState(), true)
})

test('a signed-out load keeps signedOut on the inline error', async () => {
  stubBinding(savessvc, 'Saves', () => {
    throw new Error('[signed_out] nexus')
  })
  await useSaves.getState().load('stardew', 'p1', '1')
  expect(useSaves.getState().status).toBe('error')
  expect(useSaves.getState().error?.signedOut).toBe(true)
})
