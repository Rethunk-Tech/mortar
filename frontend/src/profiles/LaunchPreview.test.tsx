import { afterAll, afterEach, beforeEach, expect, jest, mock, test } from 'bun:test'
import type { CommandPreview } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'

const { I18nProvider } = await import('@lingui/react')
const testing = await import('@testing-library/react')
const { i18n } = await import('../i18n/index.ts')
const { LaunchPreview } = await import('./LaunchPreview.tsx')
const answers = new Map<string, ReturnType<typeof Promise.withResolvers<CommandPreview>>>()
mock.module('../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts', () => ({
  PreviewCommand: (_game: string, _profile: string, options: string) => {
    const answer = Promise.withResolvers<CommandPreview>()
    answers.set(options, answer)
    return answer.promise
  },
}))

afterAll(() => {
  testing.cleanup()
})
beforeEach(() => {
  jest.useFakeTimers()
})
afterEach(() => {
  jest.useRealTimers()
})

const view = (options: string) => (
  <I18nProvider i18n={i18n}>
    <LaunchPreview gameId="stardew" profileId="p1" options={options} prefix="" env="" />
  </I18nProvider>
)

test('a slower preview of options since edited does not replace the newer one', async () => {
  const { rerender } = testing.render(view('-a'))
  testing.act(() => jest.advanceTimersByTime(300))
  rerender(view('-ab'))
  testing.act(() => jest.advanceTimersByTime(300))
  await testing.act(async () => {
    answers.get('-ab')?.resolve({ env: [], argv: ['game', '-ab'], error: '' })
    await Promise.resolve()
    answers.get('-a')?.resolve({ env: [], argv: ['game', '-a'], error: '' })
    await Promise.resolve()
  })
  expect(testing.screen.getByLabelText('Direct launch preview').textContent).toBe("'game' '-ab'")
})
