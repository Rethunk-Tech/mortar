import { afterEach, beforeEach, expect, mock, test } from 'bun:test'
import { I18nProvider } from '@lingui/react'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import * as launchsvc from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import * as settingssvc from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { i18n } from '../i18n/index.ts'
import { useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { testProfile } from '../profiles/testProfile.ts'
import { stubBinding } from '../testing/stubBinding.ts'
import { ProfileCards } from './ProfileCards.tsx'

beforeEach(() => {
  stubBinding(launchsvc, 'LastRunStarted', () => '')
  stubBinding(settingssvc, 'SetLastGame', () => undefined)
  stubBinding(settingssvc, 'SetLastProfile', () => undefined)
  useNav.setState({ route: { name: 'game-select' } })
  useProfiles.setState({ ...useProfiles.getInitialState(), load: async () => undefined }, true)
})

afterEach(() => {
  cleanup()
  mock.restore()
})

test('a profile card opens setup when the game has none yet', async () => {
  const game = { id: 'stardew', name: 'Stardew Valley', installed: false } as GameInfo
  render(
    <I18nProvider i18n={i18n}>
      <ProfileCards
        game={game}
        profiles={[testProfile({ id: 'p1', name: 'Main' })]}
        lastPlayed={undefined}
      />
    </I18nProvider>,
  )
  fireEvent.click(screen.getByRole('button', { name: /Main/ }))
  await waitFor(() => {
    expect(useNav.getState().route).toEqual({ name: 'game-setup', game: 'stardew' })
  })
})
