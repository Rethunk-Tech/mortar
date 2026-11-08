import { expect, test } from 'bun:test'
import { parseNexusKey } from './nexusKey.ts'

test('parseNexusKey reads the keys store.NexusFile reads', () => {
  expect(parseNexusKey('nexus-5098-145906')).toEqual({ modID: 5098, fileID: 145_906 })
  expect(parseNexusKey('nexus-12-34-extra')).toEqual({ modID: 12, fileID: 34 })
  expect(parseNexusKey('nexus-12')).toBeNull()
  expect(parseNexusKey('nexus--3-4')).toBeNull()
  expect(parseNexusKey('nexus-1-x')).toBeNull()
  expect(parseNexusKey('thunderstore-1-2')).toBeNull()
  expect(parseNexusKey('')).toBeNull()
})
