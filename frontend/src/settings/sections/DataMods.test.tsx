import { afterAll, afterEach, expect, mock, test } from 'bun:test'
import type { CacheInfo as CacheInfoValue } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/models.ts'

const { I18nProvider } = await import('@lingui/react')
const { act, cleanup, render, screen } = await import('@testing-library/react')
const { i18n } = await import('../../i18n/index.ts')
const datasvc = await import(
  '../../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/service.ts'
)
const { stubBinding } = await import('../../testing/stubBinding.ts')
const { CacheClearDialog } = await import('./DataMods.tsx')

afterAll(() => {
  cleanup()
})
afterEach(() => {
  mock.restore()
})

const noop = () => undefined
const dialog = (open: boolean) => (
  <I18nProvider i18n={i18n}>
    <CacheClearDialog open={open} onClose={noop} onCleared={noop} />
  </I18nProvider>
)

// The dialog outlives its close, so a size kept in state showed the previous reading until the next read landed.
test('reopening asks again and shows no size until the answer lands', async () => {
  const answers = [Promise.withResolvers<CacheInfoValue>(), Promise.withResolvers<CacheInfoValue>()]
  let asked = 0
  stubBinding(datasvc, 'CacheInfo', () => {
    const answer = answers[asked]
    asked += 1
    return answer?.promise
  })

  const view = render(dialog(true))
  await act(async () => {
    answers[0]?.resolve({ path: '/first', size: 2048 })
  })
  expect(screen.queryByText(/\/first/)).not.toBeNull()

  view.rerender(dialog(false))
  view.rerender(dialog(true))
  expect(screen.queryByText(/\/first/)).toBeNull()

  await act(async () => {
    answers[1]?.resolve({ path: '/second', size: 4096 })
  })
  expect(screen.queryByText(/\/second/)).not.toBeNull()
})
