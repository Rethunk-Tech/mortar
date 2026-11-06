import { afterEach, expect, mock, test } from 'bun:test'
import * as nexussvc from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { useNexus } from '../settings/nexus.ts'
import { stubBinding } from '../testing/stubBinding.ts'
import { loadDetails, useNexusDetails } from './nexusDetails.ts'

const before = useNexus.getState()
afterEach(() => {
  mock.restore()
  useNexus.setState(before, true)
  useNexusDetails.setState({ byKey: {} })
})

const page = { modId: 7, version: '2.0', summary: 'A mod' }

test('signed out, a mod nothing cached is known about resolves to the sign-in state, not an endless wait', async () => {
  useNexus.setState({ signedIn: false })
  stubBinding(nexussvc, 'CachedDetails', () => ({}))
  await loadDetails('swr-game', 7)
  const entry = useNexusDetails.getState().byKey['swr-game:7']
  expect(entry?.signedOut).toBe(true)
  expect(entry?.error).toBeTruthy()
  expect(entry?.details).toBeUndefined()
})

test('signed out, a saved page shows as it is, marked as one Nexus could not refresh', async () => {
  useNexus.setState({ signedIn: false })
  stubBinding(nexussvc, 'CachedDetails', () => ({ '8': { page, files: [], changelogs: [] } }))
  await loadDetails('swr-game', 8)
  const entry = useNexusDetails.getState().byKey['swr-game:8']
  expect(entry?.signedOut).toBe(true)
  expect(entry?.details?.page.version).toBe('2.0')
})
