import { expect, test } from 'bun:test'
import type { BrowseItem } from './browseTypes.ts'
import { stepSelection } from './selection.ts'

const hit = (id: string) => ({ source: 'thunderstore', id }) as BrowseItem

test('Up and Down step through the hits and stop at either end', () => {
  const items = [hit('a'), hit('b'), hit('c')]
  expect(stepSelection(items, null, 1)?.id).toBe('a')
  expect(stepSelection(items, hit('a'), 1)?.id).toBe('b')
  expect(stepSelection(items, hit('c'), 1)?.id).toBe('c')
  expect(stepSelection(items, hit('a'), -1)?.id).toBe('a')
  expect(stepSelection(items, hit('gone'), -1)?.id).toBe('a')
  expect(stepSelection([], hit('a'), 1)).toBeUndefined()
})
