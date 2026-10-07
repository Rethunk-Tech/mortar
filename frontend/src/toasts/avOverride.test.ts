import { expect, test } from 'bun:test'
import { parseDetection } from './avOverride.ts'

test('parseDetection reads the key, scanner, threat and file out of a refused install', () => {
  const e = new Error(
    'store stardew/local-abc123: [malware] the antivirus (clamd) reports Test.Threat in Mod/a.dll',
  )
  expect(parseDetection(e)).toEqual({
    key: 'local-abc123',
    scanner: 'clamd',
    name: 'Test.Threat',
    file: 'Mod/a.dll',
  })
})

test('parseDetection copes with a scanner that names no file and ignores other errors', () => {
  expect(
    parseDetection('store lethal-company/pkg-1f: [malware] the antivirus (sh) reports Win.Bad'),
  ).toEqual({ key: 'pkg-1f', scanner: 'sh', name: 'Win.Bad', file: '' })
  expect(parseDetection(new Error('disk full'))).toBeNull()
})
