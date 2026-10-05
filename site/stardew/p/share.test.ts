import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { brotliCompressSync, constants } from 'node:zlib'
import { decodeShare, type ShareError } from './share.js'

const wasm = readFileSync(
  new URL('../../vendor/brotli-dec-wasm/brotli_dec_wasm_bg.wasm', import.meta.url),
)
const encode = (v: unknown) =>
  brotliCompressSync(Buffer.from(JSON.stringify(v)), {
    params: { [constants.BROTLI_PARAM_QUALITY]: 11 },
  }).toString('base64url')

test('decodes a profile', async () => {
  const p = encode([1, 'My farm', [[2400, 99], 'owner/repo@v1.2/mod.zip']])
  expect(await decodeShare(`#${p}`, wasm)).toEqual({
    name: 'My farm',
    entries: [
      { kind: 'nexus', mod: 2400 },
      { kind: 'github', repo: 'owner/repo', tag: 'v1.2', asset: 'mod.zip' },
    ],
  })
})

test('decodes version 2 object entries', async () => {
  const p = encode([
    2,
    'My farm',
    [
      { modId: 2400, fileId: 99, disabled: ['a.b'] },
      { github: 'owner/repo@v1.2/mod.zip', note: 'x' },
      [7, 8],
    ],
  ])
  expect(await decodeShare(`#${p}`, wasm)).toEqual({
    name: 'My farm',
    entries: [
      { kind: 'nexus', mod: 2400 },
      { kind: 'github', repo: 'owner/repo', tag: 'v1.2', asset: 'mod.zip' },
      { kind: 'nexus', mod: 7 },
    ],
  })
})

test('rejects bad input', async () => {
  const kind = (h: string) => decodeShare(h, wasm).catch((e: ShareError) => e.kind)
  expect(await kind('')).toBe('missing')
  expect(await kind(`#${encode([3, 'x', []])}`)).toBe('version')
  expect(await kind(`#${encode([0, 'x', []])}`)).toBe('version')
  expect(await kind(`#${encode([2, 'x', [{ note: 'x' }]])}`)).toBe('bad')
  expect(await kind(`#${'A'.repeat(8193)}`)).toBe('bad')
  expect(await kind(`#${encode({ a: 1 })}`)).toBe('bad')
  expect(await kind(`#${encode([1, 'x', ['nope']])}`)).toBe('bad')
  expect(await kind(`#${encode([1, 'x'.repeat(70_000), []])}`)).toBe('bad')
})
