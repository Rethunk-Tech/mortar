import { describe, expect, test } from 'bun:test'
import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { nestOverlays, overlaysByBase } from './overlayRows.ts'
import type { VirtualRow } from './virtualRows.ts'

const entry = (key: string, extra: Partial<Entry> = {}) =>
  ({
    key,
    source: { kind: 'nexus', name: `${key}.zip` },
    mods: [],
    disabled: [],
    ...extra,
  }) as Entry

const base = entry('main', { mods: [{ uniqueId: 'x.main', name: 'Main' } as never] })
const profile = {
  entries: [
    base,
    entry('opt1', { overlayOf: 'main' }),
    entry('opt2', { overlayOf: 'main', overlayOff: true }),
    entry('other'),
  ],
}

describe('overlaysByBase', () => {
  test('groups optional files under their main file in profile order', () => {
    const rows = overlaysByBase(profile).get('main') ?? []
    expect(rows.map((r) => [r.key, r.label, r.enabled, r.baseEnabled])).toEqual([
      ['opt1', 'opt1.zip', true, true],
      ['opt2', 'opt2.zip', false, true],
    ])
    expect(overlaysByBase(profile).has('other')).toBe(false)
  })

  test('marks optional files of a switched-off main file', () => {
    const off = {
      entries: [{ ...base, disabled: ['x.main'] }, entry('opt1', { overlayOf: 'main' })],
    }
    expect(overlaysByBase(off).get('main')?.[0]?.baseEnabled).toBe(false)
  })
})

describe('nestOverlays', () => {
  test('nests once, right after the first row of the main file', () => {
    const rows: VirtualRow<{ key: string }>[] = [
      { kind: 'header', key: 'h:a', groupKey: 'a', count: 3 },
      { kind: 'row', key: 'r:1', groupKey: 'a', item: { key: 'main' }, stripe: false },
      { kind: 'row', key: 'r:2', groupKey: 'a', item: { key: 'main' }, stripe: true },
      { kind: 'row', key: 'r:3', groupKey: 'a', item: { key: 'other' }, stripe: false },
    ]
    const out = nestOverlays(rows, (r) => r.key, overlaysByBase(profile))
    expect(out.map((r) => r.key)).toEqual(['h:a', 'r:1', 'o:opt1', 'o:opt2', 'r:2', 'r:3'])
  })

  test('leaves rows alone when nothing is laid over them', () => {
    const rows: VirtualRow<{ key: string }>[] = [
      { kind: 'row', key: 'r:3', groupKey: '', item: { key: 'other' }, stripe: false },
    ]
    expect(nestOverlays(rows, (r) => r.key, new Map())).toEqual(rows)
  })
})
