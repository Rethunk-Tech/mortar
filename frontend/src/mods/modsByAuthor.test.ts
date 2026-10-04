import { expect, test } from 'bun:test'
import { testProfile } from '../profiles/testProfile.ts'
import { modsByAuthor } from './modsByAuthor.ts'

test('groups mods by normalised author across profiles', () => {
  const profiles = [
    testProfile({
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
    testProfile({
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
