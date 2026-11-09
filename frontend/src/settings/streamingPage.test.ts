import { expect, test } from 'bun:test'
import type { LoaderRef } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import { showsStreaming } from './streamingPage.ts'

const loader = (overlay: boolean) => ({ id: 'bepinex5', overlay }) as LoaderRef

test('the Streaming page needs a game with an overlay component', () => {
  expect(showsStreaming(loader(true))).toBe(true)
  // A bridge whose game lists no overlay component has nothing to switch.
  expect(showsStreaming(loader(false))).toBe(false)
  expect(showsStreaming(undefined)).toBe(false)
})
