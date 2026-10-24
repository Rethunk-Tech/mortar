import { afterEach, expect, mock, test } from 'bun:test'
import * as nexussvc from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { useNexus } from '../settings/nexus.ts'
import { stubBinding } from '../testing/stubBinding.ts'
import { primeDetails, useNexusDetails } from './nexusDetails.ts'

let primeCalls = 0
afterEach(() => {
  mock.restore()
})

test('priming a mod Nexus does not know settles: one request, no store churn', async () => {
  // Nexus returns nothing for 9999: hidden, removed, or a wrong id from an update key.
  stubBinding(nexussvc, 'PrimeDetails', () => {
    primeCalls += 1
    return {}
  })
  useNexus.setState({ signedIn: true } as never)
  let wakes = 0
  const unsub = useNexusDetails.subscribe(() => {
    wakes += 1
    if (wakes < 50) {
      primeDetails('stardew', [9999]).catch(() => undefined)
    }
  })
  await primeDetails('stardew', [9999])
  await primeDetails('stardew', [9999])
  await new Promise((r) => setTimeout(r, 30))
  unsub()
  expect(primeCalls).toBe(1)
  expect(wakes).toBe(0)
})
