import { expect, test } from 'bun:test'
import type { Details } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/models.ts'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { rowDetails } from './details.ts'

test('picks the new version file size and the first changelog line', () => {
  const d = {
    files: [
      { version: '1.0', modVersion: '1.0', sizeKb: 5, uploaded: '2020' },
      { version: '2.0', modVersion: '2.0', sizeKb: 9, uploaded: '2021' },
    ],
    changelogs: [{ version: '2.0', body: '\n  Fixed it\nmore', notes: [] }],
  } as unknown as Details
  expect(rowDetails({ version: '2.0' } as Update, d)).toEqual({ sizeKb: 9, changelog: 'Fixed it' })
  expect(rowDetails({ version: '3.0' } as Update, undefined)).toEqual({ sizeKb: 0, changelog: '' })
})
