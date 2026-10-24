import { afterAll, afterEach, beforeEach, expect, jest, mock, test } from 'bun:test'
import type { AssetMapPage } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/framework/contentpatcher/models.ts'

const service = '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/service.ts'
const real = await import(service)
const answers = new Map<string, ReturnType<typeof Promise.withResolvers<AssetMapPage>>>()
mock.module(service, () => ({
  ...real,
  AssetMap: (_game: string, _profile: string, filter: string) => {
    const answer = Promise.withResolvers<AssetMapPage>()
    answers.set(filter, answer)
    return answer.promise
  },
}))
const testing = await import('@testing-library/react')
const { useProfiles } = await import('../profiles/store.ts')
const { useAssetPages } = await import('./assetPages.ts')

afterAll(() => {
  testing.cleanup()
})
beforeEach(() => {
  jest.useFakeTimers()
  useProfiles.setState({ game: { id: 'stardew' } as never, openId: 'p1' })
})
afterEach(() => {
  jest.useRealTimers()
})

const page = (target: string) =>
  ({ all: 1, shared: 1, total: 1, targets: [{ target }] }) as unknown as AssetMapPage

test('a slower page for a filter since edited does not replace the newer one', async () => {
  const { result, rerender } = testing.renderHook(({ q }) => useAssetPages(true, q, true), {
    initialProps: { q: 'a' },
  })
  testing.act(() => jest.advanceTimersByTime(300))
  rerender({ q: 'ab' })
  testing.act(() => jest.advanceTimersByTime(300))
  await testing.act(async () => {
    answers.get('ab')?.resolve(page('Maps/ab'))
    await Promise.resolve()
    answers.get('a')?.resolve(page('Maps/a'))
    await Promise.resolve()
  })
  expect(result.current.page?.targets?.[0]?.target).toBe('Maps/ab')
})
