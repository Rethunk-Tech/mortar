import { afterAll, expect, mock, test } from 'bun:test'
import type { CrashHint as Hint } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'

const service = '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
const real = await import(service)
const answers = new Map<string, ReturnType<typeof Promise.withResolvers<Hint | null>>>()
mock.module(service, () => ({
  ...real,
  CrashHint: (_game: string, profile: string) => {
    const answer = Promise.withResolvers<Hint | null>()
    answers.set(profile, answer)
    return answer.promise
  },
}))
const testing = await import('@testing-library/react')
const { useCrashHint } = await import('./crashHint.ts')

afterAll(() => {
  testing.cleanup()
})

// The card offers to switch off the mod the hint blames: another profile's hint must not offer that here.
test("a profile opened after another shows none of the other's crash hint", async () => {
  const { result, rerender } = testing.renderHook(({ id }) => useCrashHint('stardew', id, '1'), {
    initialProps: { id: 'p1' },
  })
  await testing.act(async () => {
    answers.get('p1')?.resolve({ modKey: 'nexus-1-2' } as Hint)
    await Promise.resolve()
  })
  expect(result.current?.modKey).toBe('nexus-1-2')
  rerender({ id: 'p2' })
  expect(result.current).toBeNull()
})
