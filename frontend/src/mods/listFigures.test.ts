import { expect, test } from 'bun:test'
import type { Details } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/models.ts'
import type {
  Entry,
  Mod,
  ModFact,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { ListRow } from './listColumns.ts'
import { figureFields, figureNumber, figureText, keepSameFacts } from './listFigures.ts'

const fact = (over: Partial<ModFact>): ModFact => ({
  kind: '',
  released: '',
  latest: '',
  latestReleased: '',
  updated: '',
  rating: 0,
  downloads: 0,
  ...over,
})

const row = (fields: Pick<ListRow, 'facts' | 'figures'>, endorsements = 0): ListRow => ({
  mod: { id: 'thunderstore:Ns-Foo', version: '1.0.0', endorsements } as Mod,
  added: '',
  switchedOff: '',
  source: '',
  status: '',
  note: '',
  tags: [],
  categoryLabel: '',
  reason: '',
  ...fields,
})

test('a Nexus mod is dated by the file it installed, and a Thunderstore mod by its listing', () => {
  const entry = { source: { kind: 'nexus', fileId: 7 } } as Entry
  const details = {
    page: { version: '2.0.0', updated: '2026-02-01', endorsements: 10, downloads: 100 },
    files: [{ fileId: 7, uploaded: '2026-01-05' }],
  } as Details
  expect(figureFields(entry, details, undefined).figures).toEqual({
    latest: '2.0.0',
    released: '2026-01-05',
    updated: '2026-02-01',
    rating: 10,
    downloads: 100,
  })
  const ts = fact({
    latest: '1.2.0',
    released: '2025-12-01',
    updated: '2026-01-01',
    rating: 5,
    downloads: 9,
  })
  expect(figureFields(undefined, undefined, ts).figures?.released).toBe('2025-12-01')
  expect(figureFields(undefined, undefined, fact({ kind: 'plugin' })).figures).toBeUndefined()
})

test('the id column names the plugin GUIDs, not Mortar id for the package', () => {
  const r = row(
    figureFields(undefined, undefined, fact({ ids: ['com.ns.foo', 'com.ns.bar'], kind: 'plugin' })),
  )
  expect(figureText(r, 'id')).toBe('com.ns.foo, com.ns.bar')
  expect(figureText(r, 'kind')).toBe('plugin')
  expect(figureText(row({}), 'id')).toBe('')
})

test('a mod with endorsements and no figures keeps them; downloads need figures', () => {
  expect(figureNumber(row({}, 4), 'endorsements')).toBe(4)
  expect(figureNumber(row({}), 'endorsements')).toBeUndefined()
  expect(figureNumber(row({}, 4), 'downloads')).toBeUndefined()
})

test('a re-read keeps the facts that read the same, and the whole map when none changed', () => {
  const a = fact({ latest: '1.0.0' })
  const b = fact({ latest: '2.0.0' })
  const prev = { a, b }
  expect(keepSameFacts(prev, { a: fact({ latest: '1.0.0' }), b: fact({ latest: '2.0.0' }) })).toBe(
    prev,
  )
  const next = keepSameFacts(prev, { a: fact({ latest: '1.0.0' }), b: fact({ latest: '2.1.0' }) })
  expect(next).not.toBe(prev)
  expect(next.a).toBe(a)
  expect(next.b?.latest).toBe('2.1.0')
  expect(keepSameFacts(prev, { a: fact({ latest: '1.0.0' }) })).toEqual({ a })
})
