import { afterEach, beforeEach, expect, mock, test } from 'bun:test'
import { I18nProvider } from '@lingui/react'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import * as launchsvc from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { i18n } from '../i18n/index.ts'
import { useLaunch } from '../launch/store.ts'
import { stubBinding } from '../testing/stubBinding.ts'
import { LaunchOptionsBlock } from './LaunchOptionsBlock.tsx'

const noop = () => undefined

let release: (value: { reachedTitle: boolean; cause: string }) => void = noop

beforeEach(() => {
  cleanup()
  useLaunch.setState(useLaunch.getInitialState(), true)
  stubBinding(launchsvc, 'PreviewCommand', () => ({ env: [], argv: [], error: '' }))
  stubBinding(launchsvc, 'TestLaunchWith', () => {
    return new Promise<{ reachedTitle: boolean; cause: string }>((resolve) => {
      release = resolve
    })
  })
})

afterEach(() => {
  cleanup()
  mock.restore()
})

test('Test launch ignores a second click while the first is in flight', async () => {
  render(
    <I18nProvider i18n={i18n}>
      <LaunchOptionsBlock
        gameId="stardewvalley"
        profileId="default"
        launchOptions=""
        onLaunchOptions={noop}
        launchPrefix=""
        onLaunchPrefix={noop}
        launchEnv=""
        onLaunchEnv={noop}
        launchError={null}
        onLaunchError={noop}
      />
    </I18nProvider>,
  )
  const button = screen.getByRole('button', { name: /test launch/i })
  expect((button as HTMLButtonElement).disabled).toBe(false)
  fireEvent.click(button)
  fireEvent.click(button)
  expect(launchsvc.TestLaunchWith).toHaveBeenCalledTimes(1)
  release({ reachedTitle: true, cause: '' })
  await waitFor(() => {
    expect(screen.getByText(/reached the title screen/i)).toBeTruthy()
  })
})
