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
