import { afterAll, expect, test } from 'bun:test'

const testing = await import('@testing-library/react')
const { useCategoryNames } = await import('./useCategoryNames.ts')

afterAll(() => {
  testing.cleanup()
})

test('hands the filter row the same object until the names change', async () => {
  const categories = () => Promise.resolve(['Items'])
  const { result, rerender } = testing.renderHook(() =>
    useCategoryNames({ categories, game: 'g', source: 's', skip: false }),
  )
  await testing.act(async () => new Promise<void>((resolve) => setTimeout(resolve, 0)))
  expect(result.current.names).toEqual(['Items'])
  const before = result.current
  rerender()
  expect(result.current).toBe(before)
})
