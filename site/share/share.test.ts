import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { brotliCompressSync, constants } from 'node:zlib'
import { decodeShare, newer, type ShareError } from './share.js'

const wasm = readFileSync(
  new URL('../vendor/brotli-dec-wasm/brotli_dec_wasm_bg.wasm', import.meta.url),
)
const encode = (v: unknown) =>
  brotliCompressSync(Buffer.from(JSON.stringify(v)), {
    params: { [constants.BROTLI_PARAM_QUALITY]: 11 },
  }).toString('base64url')

const keys = { nexus: 'stardewvalley', thunderstore: 'lethal-company' }

test('decodes a version 3 profile', async () => {
  const p = encode([
    3,
    'My farm',
    'stardew',
    keys,
    [
      { s: 'nexus', mod: 2400, file: 99, disabled: ['a.b'] },
      { s: 'github', repo: 'owner/repo', tag: 'v1.2', asset: 'mod.zip', note: 'x' },
      { s: 'thunderstore', ns: 'Alice', name: 'MoreCompany', version: '1.2.3' },
    ],
  ])
  expect(await decodeShare(`#${p}`, wasm)).toEqual({
    name: 'My farm',
    game: 'stardew',
    gameVersion: '',
    sourceKeys: keys,
    entries: [
      { kind: 'nexus', mod: 2400 },
      { kind: 'github', repo: 'owner/repo', tag: 'v1.2', asset: 'mod.zip' },
      { kind: 'thunderstore', ns: 'Alice', name: 'MoreCompany', version: '1.2.3' },
    ],
  })
})

test('decodes a Lethal Company profile of Thunderstore packages', async () => {
  const lc = { thunderstore: 'lethal-company' }
  const p = encode([
    3,
    'Lobby',
    'lethal-company',
    lc,
    [{ s: 'thunderstore', ns: 'x753', name: 'More_Suits', version: '1.5.4' }],
  ])
  expect(await decodeShare(`#${p}`, wasm)).toEqual({
    name: 'Lobby',
    game: 'lethal-company',
    gameVersion: '',
    sourceKeys: lc,
    entries: [{ kind: 'thunderstore', ns: 'x753', name: 'More_Suits', version: '1.5.4' }],
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
  expect(
    await kind(`#${encode([3, 'x', 'stardew', {}, [{ s: 'thunderstore', ns: 'A', name: 'B' }]])}`),
  ).toBe('bad')
  expect(
    await kind(
      `#${encode([3, 'x', 'stardew', keys, [{ s: 'thunderstore', ns: 'A/..', name: 'B' }]])}`,
    ),
  ).toBe('bad')
  expect(await kind(`#${'A'.repeat(8193)}`)).toBe('bad')
  expect(await kind(`#${encode({ a: 1 })}`)).toBe('bad')
  expect(await kind(`#${encode([3, 'x', 'stardew', keys, ['nope']])}`)).toBe('bad')
  expect(await kind(`#${encode([3, 'x'.repeat(70_000), 'stardew', keys, []])}`)).toBe('bad')
})

test('reads the game version and each mod size and minimum game version', async () => {
  const p = encode([
    3,
    'Main',
    'stardew',
    keys,
    [
      { s: 'nexus', mod: 1, file: 2, kb: 12_000, min: '1.6.15' },
      { s: 'thunderstore', ns: 'A', name: 'B', version: '1.0.0', kb: -1, min: 'soon' },
    ],
    '1.6.9',
  ])
  const share = await decodeShare(`#${p}`, wasm)
  expect(share.gameVersion).toBe('1.6.9')
  expect(share.entries).toEqual([
    { kind: 'nexus', mod: 1, kb: 12_000, min: '1.6.15' },
    { kind: 'thunderstore', ns: 'A', name: 'B', version: '1.0.0' },
  ])
  expect(newer('1.6.15', '1.6.9')).toBe(true)
  expect(newer('1.6', '1.6.0')).toBe(false)
  expect(newer('1.5.6', '1.6')).toBe(false)
})
