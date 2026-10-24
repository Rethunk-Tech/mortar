import { afterAll, expect, mock, test } from 'bun:test'
import type { HistoryDiff } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

mock.module('@lingui/core/macro', () => ({
  msg: (parts: TemplateStringsArray, ...values: unknown[]) => String.raw({ raw: parts }, ...values),
}))
const testing = await import('@testing-library/react')
const { useProfiles } = await import('./store.ts')
const { useHistoryPanel } = await import('./useHistoryPanel.ts')
// Mocked after the import: the profile store reads other exports of this module, which a mock first would hide.
const answers = new Map<string, ReturnType<typeof Promise.withResolvers<HistoryDiff>>>()
mock.module('../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts', () => ({
  History: async () => [],
  EventDiff: async () => ({ items: [] }),
  HistoryDiff: (_game: string, _profile: string, a: string, b: string) => {
    const answer = Promise.withResolvers<HistoryDiff>()
    answers.set(`${a},${b}`, answer)
    return answer.promise
  },
}))

afterAll(() => {
  testing.cleanup()
})

const diff = (a: string, b: string) => ({ a, b, items: [] }) as unknown as HistoryDiff

test('a comparison answered after the picks changed is dropped', async () => {
  useProfiles.setState({ game: { id: 'stardew' } as never })
  const { result } = testing.renderHook(() => useHistoryPanel('p1', false))
  testing.act(() => result.current.setPicked(['a', 'b']))
  testing.act(() => result.current.setPicked(['a', 'c']))
  await testing.act(async () => {
    answers.get('a,c')?.resolve(diff('a', 'c'))
    await Promise.resolve()
    answers.get('a,b')?.resolve(diff('a', 'b'))
    await Promise.resolve()
  })
  expect(result.current.pair?.b).toBe('c')
})
