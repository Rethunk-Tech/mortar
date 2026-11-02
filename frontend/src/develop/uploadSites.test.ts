import { expect, test } from 'bun:test'
import { tokenToForget } from './uploadSites.ts'

test('only sites that keep an upload token have one to forget', () => {
  expect(['thunderstore', 'curseforge', 'nexus', 'github', 'other'].map(tokenToForget)).toEqual([
    'thunderstore',
    'curseforge',
    undefined,
    undefined,
    undefined,
  ])
})
