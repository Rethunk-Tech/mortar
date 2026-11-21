import { afterEach, expect, mock, spyOn, test } from 'bun:test'
import { I18nProvider } from '@lingui/react'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { Clipboard } from '@wailsio/runtime'
import * as sharesvc from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/service.ts'
import { i18n } from '../i18n/index.ts'
import { stubBinding } from '../testing/stubBinding.ts'
import { HostedLink } from './HostedLink.tsx'
import type { ShownInfo } from './logic.ts'
import { shareIncludeDefaults } from './shareDefaults.ts'

afterEach(() => {
  cleanup()
  mock.restore()
})

const include = shareIncludeDefaults({})

function info(hosted: ShownInfo['hosted'] = null): ShownInfo {
  return {
    name: 'Farm',
    web: 'https://mortar.rethunk.tech/stardew/p#abc',
    app: 'mortar://stardew/p/abc',
    length: 40,
    limit: 2000,
    count: 1,
    mods: 1,
    tooLarge: false,
    groups: [],
    leftOut: [],
    hosted,
  }
}

function mount(hosted: ShownInfo['hosted'] = null, ready = true) {
  stubBinding(sharesvc, 'HostedSharesReady', () => ready)
  spyOn(Clipboard, 'SetText').mockImplementation(() => Promise.resolve() as never)
  return render(
    <I18nProvider i18n={i18n}>
      <HostedLink
        game="stardew"
        profileId="0123456789abcdef"
        keys={[]}
        include={include}
        info={info(hosted)}
      />
    </I18nProvider>,
  )
}

test('Get short link posts the share and shows the hosted URL', async () => {
  const created = {
    code: 'AbCdEfGh12',
    version: 1,
    url: 'https://mortar.rethunk.tech/stardew/s/AbCdEfGh12',
  }
  stubBinding(sharesvc, 'CreateHostedShare', () => created)
  mount()
  expect(await screen.findByRole('button', { name: 'Get short link' })).toBeTruthy()
  expect(
    screen.getByText(
      'A short link stores the mod list on mortar.rethunk.tech. The long link keeps it private.',
    ),
  ).toBeTruthy()
  fireEvent.click(screen.getByRole('button', { name: 'Get short link' }))
  await waitFor(() => {
    expect(screen.getByText(created.url)).toBeTruthy()
  })
  expect(screen.getByRole('button', { name: 'Update short link' })).toBeTruthy()
  expect(screen.getByText('Version 1')).toBeTruthy()
})

test('Update short link is offered when the profile already has one', async () => {
  stubBinding(sharesvc, 'UpdateHostedShare', () => ({
    code: 'AbCdEfGh12',
    version: 2,
    url: 'https://mortar.rethunk.tech/stardew/s/AbCdEfGh12',
  }))
  mount({
    code: 'AbCdEfGh12',
    version: 1,
    url: 'https://mortar.rethunk.tech/stardew/s/AbCdEfGh12',
  })
  expect(await screen.findByRole('button', { name: 'Update short link' })).toBeTruthy()
  fireEvent.click(screen.getByRole('button', { name: 'Update short link' }))
  await waitFor(() => {
    expect(screen.getByText('Version 2')).toBeTruthy()
  })
})

test('No short link is offered while the share service does not answer', async () => {
  mount(null, false)
  await waitFor(() => {
    expect(screen.queryByRole('button', { name: 'Get short link' })).toBeNull()
  })
  expect(screen.queryByText(/A short link stores/)).toBeNull()
})
