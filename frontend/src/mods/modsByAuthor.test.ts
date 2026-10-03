import { expect, test } from 'bun:test'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { modsByAuthor } from './modsByAuthor.ts'

const profile = (partial: Partial<Profile> & Pick<Profile, 'id' | 'name'>): Profile => ({
  notes: '',
  cover: '',
  order: 0,
  hidden: false,
  created: '',
  updated: '',
  entries: null,
  ...partial,
})

test('groups mods by normalised author across profiles', () => {
  const profiles = [
    profile({
      id: 'a',
      name: 'Farm',
      entries: [
        {
          key: 'k1',
          previousKey: '',
          source: { kind: 'local', name: 'x' },
          mods: [
            {
              uniqueId: 'Author.One',
              name: 'One',
              version: '1',
              author: 'Pathoschild & Helper',
              folder: '.',
              needs: null,
            },
          ],
          disabled: [],
          added: '',
        },
      ],
    }),
    profile({
      id: 'b',
      name: 'Co-op',
      entries: [
        {
          key: 'k2',
          previousKey: '',
          source: { kind: 'local', name: 'y' },
          mods: [
            {
              uniqueId: 'Author.Two',
              name: 'Two',
              version: '2',
              author: 'pathoschild',
              folder: '.',
              needs: null,
            },
          ],
          disabled: ['Author.Two'],
          added: '',
        },
      ],
    }),
  ]
  const rows = modsByAuthor(profiles, 'Pathoschild')
  expect(
    rows
      .map((r) => r.uniqueId)
      .toSorted((a, b) => a.localeCompare(b, undefined, { sensitivity: 'base' })),
  ).toEqual(['Author.One', 'Author.Two'])
  const two = rows.find((r) => r.uniqueId === 'Author.Two')
  expect(two?.profiles[0]?.enabled).toBe(false)
})
