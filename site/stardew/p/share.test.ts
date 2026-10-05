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

const keys = { nexus: 'stardewvalley' }

test('decodes a version 3 profile', async () => {
  const p = encode([
    3,
    'My farm',
    'stardew',
    keys,
    [
      { s: 'nexus', mod: 2400, file: 99, disabled: ['a.b'] },
      { s: 'github', repo: 'owner/repo', tag: 'v1.2', asset: 'mod.zip', note: 'x' },
    ],
  ])
  expect(await decodeShare(`#${p}`, wasm)).toEqual({
    name: 'My farm',
    game: 'stardew',
    sourceKeys: keys,
    entries: [
      { kind: 'nexus', mod: 2400 },
      { kind: 'github', repo: 'owner/repo', tag: 'v1.2', asset: 'mod.zip' },
    ],
  })
})

test('rejects bad input', async () => {
  const kind = (h: string) => decodeShare(h, wasm).catch((e: ShareError) => e.kind)
  expect(await kind('')).toBe('missing')
  expect(await kind(`#${encode([4, 'x', 'stardew', keys, []])}`)).toBe('version')
  expect(await kind(`#${encode([2, 'x', []])}`)).toBe('bad')
  expect(await kind(`#${encode([2, 'x', 'stardew', keys, []])}`)).toBe('bad')
  expect(await kind(`#${encode([0, 'x', 'stardew', keys, []])}`)).toBe('bad')
  expect(await kind(`#${encode([3, 'x', 'stardew', {}, [{ s: 'nexus', mod: 1, file: 2 }]])}`)).toBe(
    'bad',
  )
  expect(await kind(`#${encode([3, 'x', 'stardew', keys, [{ note: 'x' }]])}`)).toBe('bad')
  expect(await kind(`#${encode([3, 'x', 'stardew', keys, [[1, 2]]])}`)).toBe('bad')
  expect(await kind(`#${'A'.repeat(8193)}`)).toBe('bad')
  expect(await kind(`#${encode({ a: 1 })}`)).toBe('bad')
  expect(await kind(`#${encode([3, 'x', 'stardew', keys, ['nope']])}`)).toBe('bad')
  expect(await kind(`#${encode([3, 'x'.repeat(70_000), 'stardew', keys, []])}`)).toBe('bad')
})
