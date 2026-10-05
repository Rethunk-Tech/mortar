import { expect, test } from 'bun:test'
import type { StarterTemplate } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/components/models.ts'
import { starterWants } from './starterWants.ts'

test('each package becomes the download its source expects', () => {
  const template = {
    id: 't',
    name: 'T',
    description: '',
    packages: [
      { source: 'nexus', ref: '541', name: 'Lookup Anything' },
      { source: 'github', ref: 'o/r', name: 'R' },
      { source: 'thunderstore', ref: 'Ns-Name', name: 'Name' },
      { source: 'modrinth', ref: 'slug', name: 'Slug' },
    ],
  } as StarterTemplate
  expect(starterWants(template)).toEqual([
    { kind: 'install', name: 'Lookup Anything', modId: 541, latest: true },
    { kind: 'install', name: 'R', repo: 'o/r' },
    { kind: 'install', name: 'Name', package: 'Ns-Name' },
    { kind: 'install', name: 'Slug', source: 'modrinth', package: 'slug' },
  ])
})
