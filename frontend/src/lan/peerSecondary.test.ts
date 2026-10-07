import { expect, test } from 'bun:test'
import { peerSecondary } from './peerSecondary.ts'

const paired = 'Paired · sends mod files'
const unpaired = 'Not paired · they download each mod'

test('secondary text follows pairing and prefixes the id for shared names', () => {
  expect(peerSecondary({ id: 'a:1', paired: true }, false, paired, unpaired)).toBe(paired)
  expect(peerSecondary({ id: 'a:1', paired: false }, false, paired, unpaired)).toBe(unpaired)
  expect(peerSecondary({ id: 'a:1', paired: false }, true, paired, unpaired)).toBe(
    `a:1 · ${unpaired}`,
  )
})
