import { afterAll, expect, test } from 'bun:test'
import { GlobalRegistrator } from '@happy-dom/global-registrator'

// A real DOM for the real MUI menus; registered before anything that touches `document` is imported.
GlobalRegistrator.register({ url: 'http://localhost/' })
const games = [
  { id: 'stardew', name: 'Stardew Valley', available: true, installed: true },
  { id: 'lethal-company', name: 'Lethal Company', available: true, installed: true },
]
// Every Wails binding call answers with the game list, which is also a non-empty profile list for the setup check.
globalThis.fetch = (() => Promise.resolve(Response.json(games))) as unknown as typeof fetch

const { I18nProvider } = await import('@lingui/react')
const { cleanup, render, screen, waitFor } = await import('@testing-library/react')
const { default: userEvent } = await import('@testing-library/user-event')
const { GameMenu } = await import('../games/GameMenu.tsx')
const { i18n } = await import('../i18n/index.ts')
const { useNav } = await import('../nav/store.ts')
const { AppMenu } = await import('./AppMenu.tsx')

afterAll(async () => {
  cleanup()
  await GlobalRegistrator.unregister()
})

function bar() {
  return render(
    <I18nProvider i18n={i18n}>
      <GameMenu />
      <AppMenu />
    </I18nProvider>,
  )
}

test('choosing another game in the Games menu opens it', async () => {
  useNav.setState({ route: { name: 'game', game: 'stardew' } })
  const user = userEvent.setup()
  bar()
  await user.click(await screen.findByRole('button', { name: /^Switch game/ }))
  const row = await screen.findByRole('menuitemradio', { name: 'Lethal Company' })
  expect(row.querySelector('[aria-hidden="true"]')).not.toBeNull()
  await user.click(row)
  await waitFor(() =>
    expect(useNav.getState().route).toEqual({ name: 'game', game: 'lethal-company' }),
  )
  cleanup()
})

test('a click on another title bar trigger while a menu is open switches menus at once', async () => {
  useNav.setState({ route: { name: 'game', game: 'stardew' } })
  const user = userEvent.setup()
  bar()
  await user.click(await screen.findByRole('button', { name: /^Switch game/ }))
  await screen.findByRole('menu', { name: 'Games' })
  await user.click(document.querySelector('[aria-label="Mortar menu"]') as HTMLElement)
  await screen.findByRole('menu', { name: 'Mortar' })
  expect(screen.queryByRole('menu', { name: 'Games' })).toBeNull()
  cleanup()
})
