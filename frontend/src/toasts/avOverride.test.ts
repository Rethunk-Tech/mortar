import { expect, test } from 'bun:test'
import { detectionOf } from './avOverride.ts'

const detail = {
  game: 'stardew',
  key: 'local-abc123',
  scanner: 'clamd',
  name: 'Test.Threat',
  file: 'Mod/a.dll',
}

test('detectionOf reads the typed detail a malware error carries', () => {
  const e = new Error('[malware] the antivirus (clamd) reports Test.Threat', {
    cause: { kind: 'malware', detail },
  })
  expect(detectionOf(e)).toEqual(detail)
})

test('detectionOf ignores errors without a complete detail', () => {
  expect(detectionOf(new Error('disk full', { cause: { kind: 'disk_full' } }))).toBeNull()
  expect(
    detectionOf(new Error('x', { cause: { kind: 'malware', detail: { key: 'k' } } })),
  ).toBeNull()
  expect(detectionOf('store a/b: nope')).toBeNull()
})
