import { afterEach, expect, mock, test } from 'bun:test'
import { I18nProvider } from '@lingui/react'
import { cleanup, render } from '@testing-library/react'
import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import * as savessvc from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/service.ts'
import { i18n } from '../i18n/index.ts'
import { useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { testProfile } from '../profiles/testProfile.ts'
import { stubBinding } from '../testing/stubBinding.ts'
import { Detail } from './Detail.tsx'

afterEach(() => {
  cleanup()
  mock.restore()
  useProfiles.setState(useProfiles.getInitialState(), true)
})

test('a game just switched to asks for no profile until its own profiles load', () => {
  const asked: string[] = []
  stubBinding(savessvc, 'Saves', (game: string, profile: string) => {
    asked.push(`${game}/${profile}`)
    return []
  })
  useNav.setState({ route: { name: 'game', game: 'stardew' } })
  useProfiles.setState({
    ...useProfiles.getInitialState(),
    game: { id: 'lethal-company' } as GameInfo,
    profiles: [testProfile({ id: 'lc1', name: 'Lobby' })],
    openId: 'lc1',
    loaded: true,
    load: async () => undefined,
  })
  render(
    <I18nProvider i18n={i18n}>
      <Detail />
    </I18nProvider>,
  )
  expect(asked).toEqual([])
})
