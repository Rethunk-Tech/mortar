import { afterEach, expect, mock, test } from 'bun:test'
import { I18nProvider } from '@lingui/react'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import * as gamesvc from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'
import * as picker from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { i18n } from '../i18n/index.ts'
import { stubBinding } from '../testing/stubBinding.ts'
import { AddGameDialog } from './AddGameDialog.tsx'

afterEach(() => {
  cleanup()
  mock.restore()
})

test('GOG Graveyard Keeper warns that plugins will not load and still lets Add run', async () => {
  stubBinding(gamesvc, 'List', () => [])
  stubBinding(gamesvc, 'CustomLoaders', () => [{ id: 'bepinex5', name: 'BepInEx 5' }])
  stubBinding(picker, 'PickExecutable', () => '/gog/Graveyard Keeper/Graveyard Keeper.exe')
  stubBinding(gamesvc, 'GuessCustomGame', () => ({
    name: 'Graveyard Keeper',
    loader: 'bepinex5',
    warning: 'gog-32bit',
  }))

  render(
    <I18nProvider i18n={i18n}>
      <AddGameDialog open={true} onClose={() => undefined} />
    </I18nProvider>,
  )
  fireEvent.click(screen.getByRole('button', { name: /Choose the game's program/ }))
  await waitFor(() => {
    expect(screen.getByText(/plugins will not load/)).toBeTruthy()
  })
  expect((screen.getByRole('button', { name: /^Add$/ }) as HTMLButtonElement).disabled).toBe(false)
})
