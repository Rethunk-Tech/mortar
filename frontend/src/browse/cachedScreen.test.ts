import { expect, test } from 'bun:test'

import type { BrowseItem } from './browseTypes.ts'
import { type BrowseResult, cachedScreen } from './useBrowseQuery.ts'

const page = (from: number): BrowseResult => ({
  total: 60,
  items: Array.from({ length: 20 }, (_, i) => ({ id: String(from + i) }) as BrowseItem),
})

test('a screen shows from answered pages only when every page it spans has answered', () => {
  const done = new Map([[1, page(0)]])
  expect(cachedScreen(done, 1, 24)).toBeNull()
  done.set(2, page(20))
  const screen = cachedScreen(done, 1, 24)
  expect(screen?.next.items.map((i) => i.id)).toEqual(
    Array.from({ length: 24 }, (_, i) => String(i)),
  )
  expect(cachedScreen(done, 2, 24)?.next.items[0]?.id).toBeUndefined()
})

// A single source's answer carries pages 0 (BrowseHost reads its missing count as 0): a screen still takes every page it
// spans, not the first alone, so a wide window's 42-hit screen is full.
test('a screen spans every source page when the answer has no page count', () => {
  const single = (from: number): BrowseResult => ({ ...page(from), total: 51_186, pages: 0 })
  const done = new Map([
    [1, single(0)],
    [2, single(20)],
    [3, single(40)],
  ])
  expect(cachedScreen(done, 1, 42)?.next.items).toHaveLength(42)
})
