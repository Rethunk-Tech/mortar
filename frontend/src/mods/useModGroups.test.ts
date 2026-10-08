import { expect, mock, test } from 'bun:test'
import { i18n } from '@lingui/core'
import type { AssetConflict } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/framework/models.ts'
import type {
  Result,
  UpdatesResult,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type {
  Entry,
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

mock.module('@lingui/core/macro', () => ({
  msg: (parts: TemplateStringsArray, ...values: unknown[]) => String.raw({ raw: parts }, ...values),
  plural: (n: number, forms: { other: string }) => forms.other.replace('#', String(n)),
}))
i18n.load('en', {})
i18n.activate('en')

const { customCategoryById } = await import('./group.ts')
const { DEFAULT_LIST_COLUMN_SORT } = await import('./listColumns.ts')
const { modId } = await import('./lookup.ts')
const { buildModGroups } = await import('./useModGroups.ts')
const { flattenModGroups } = await import('./virtualRows.ts')

// A profile the size of the maintainer's: 671 entries holding 811 mods, 40 asset conflicts and 30 updates.
const entries: Entry[] = []
const mods: Mod[] = []
for (let e = 0; e < 671; e++) {
  const key = `nexus-${10_000 + e}-${200_000 + e}`
  const ids = Array.from({ length: e < 140 ? 2 : 1 }, (_, m) => `smapi:Fixture.Mod${e}_${m}`)
  entries.push({
    key,
    previousKey: '',
    source: { kind: 'nexus', name: `Fixture ${e}.zip`, modId: 10_000 + e, fileId: 200_000 + e },
    mods: ids.map((id) => ({ id, version: '1.0.0', name: id, author: 'Fixture', folder: id })),
    disabled: e % 37 === 0 ? ids : [],
    tags: e % 9 === 0 ? ['Farm'] : [],
  })
  for (const id of ids) {
    mods.push({
      key,
      id,
      name: `Fixture Mod ${e} ${id.length}`,
      author: `Author ${e % 60}`,
      version: '1.0.0',
      enabled: e % 37 !== 0,
      picture: '',
      endorsements: e * 13,
      needs: e > 0 ? [mods[(e * 7) % mods.length]?.id ?? ''] : [],
      contentPackFor: e % 2 === 0 ? 'smapi:Pathoschild.ContentPatcher' : '',
    })
  }
}
const profile: Profile = {
  id: 'p',
  name: 'Fixture',
  notes: '',
  cover: '',
  order: 0,
  created: '',
  updated: '',
  entries,
}
const conflict = (i: number): AssetConflict => ({
  kind: 'edit',
  target: `Data/Target${i}`,
  packIds: [0, 1, 2, 3, 4].map((j) => mods[(i * 17 + j * 101) % mods.length]?.id ?? ''),
  names: [],
  keys: [],
  winnerId: '',
  winnerKind: '',
  winnerName: '',
  overridden: [],
  cosmetic: false,
  fixes: [],
  evidence: [],
})
const problems: Result = {
  missing: [],
  duplicates: [],
  broken: [],
  assetConflicts: Array.from({ length: 40 }, (_, i) => conflict(i)),
  settings: [],
  runErrors: [],
  dismissed: [],
  unknown: false,
}
const updates: UpdatesResult = {
  updates: mods.slice(0, 30).map((m) => ({
    key: m.key,
    id: m.id,
    name: m.name,
    installed: '1.0.0',
    version: '1.1.0',
    url: '',
    nexusId: 0,
    githubRepo: '',
    unofficial: false,
    source: 'nexus',
  })),
  held: [],
  unknown: false,
}
const data = {
  byId: {},
  customById: customCategoryById([]),
  sizes: {},
  costs: {},
  wins: {},
}

// About twice the CPU time five groupings took when the budget was set (75 ms); CPU, not wall, so a busy machine does
// not fail it. Held only under MORTAR_PERF=1 (`bun run perf`), like the Go budgets: a shared CI runner's CPU is slower
// than the machine the budget was set on.
const BUDGET_MS = 150
const holdBudget = process.env.MORTAR_PERF === '1'

test('the Mods list builds its rows for an 811-mod profile within budget', () => {
  const build = () => {
    for (const groupBy of ['status', 'category', 'source', 'author', 'none'] as const) {
      const groups = buildModGroups(mods, profile, data, {
        groupBy,
        sort: DEFAULT_LIST_COLUMN_SORT,
        problems,
        updates,
      })
      const rows = flattenModGroups(groups, {
        grouped: groupBy !== 'none',
        collapsed: {},
        idOf: (r) => modId(r.mod),
      })
      expect(rows.filter((r) => r.kind === 'row')).toHaveLength(mods.length)
    }
  }
  build()
  if (!holdBudget) {
    return
  }
  let best = Number.POSITIVE_INFINITY
  for (let i = 0; i < 3; i++) {
    const start = process.cpuUsage()
    build()
    const used = process.cpuUsage(start)
    best = Math.min(best, (used.user + used.system) / 1000)
  }
  expect(best).toBeLessThan(BUDGET_MS)
})
