import { expect, mock, test } from 'bun:test'
import type { Backup } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/backup/models.ts'

const testing = await import('@testing-library/react')
const answers = new Map<string, ReturnType<typeof Promise.withResolvers<Backup[]>>>()
mock.module('../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/service.ts', () => ({
  ListBackups: (game: string) => {
    const answer = Promise.withResolvers<Backup[]>()
    answers.set(game, answer)
    return answer.promise
  },
}))
const { useAllBackups } = await import('./useAllBackups.ts')

test("a slower answer for the game shown before does not replace the current game's backups", async () => {
  const { result, rerender } = testing.renderHook(({ game }) => useAllBackups(game, 's'), {
    initialProps: { game: 'stardew' },
  })
  rerender({ game: 'lethalcompany' })
  await testing.act(async () => {
    answers.get('lethalcompany')?.resolve([{ name: 'lc' } as Backup])
    await Promise.resolve()
    answers.get('stardew')?.resolve([{ name: 'sv' } as Backup])
    await Promise.resolve()
  })
  expect(result.current.all?.map((b) => b.name)).toEqual(['lc'])
})
