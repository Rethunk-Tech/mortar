import { afterEach, expect, test } from 'bun:test'
import { I18nProvider } from '@lingui/react'
import { cleanup, render, screen } from '@testing-library/react'
import { i18n } from '../i18n/index.ts'
import { EditProfileDialog } from './EditProfileDialog.tsx'
import { useProfiles } from './store.ts'
import { testProfile } from './testProfile.ts'

afterEach(() => {
  cleanup()
  useProfiles.setState(useProfiles.getInitialState(), true)
})

const profile = testProfile({ id: 'p', name: 'Main' })

test('reopening Edit profile starts on the tab the caller asked for', () => {
  const view = (open: boolean, initialTab?: 'appearance' | 'launch') => (
    <I18nProvider i18n={i18n}>
      <EditProfileDialog
        profile={profile}
        open={open}
        onClose={() => undefined}
        {...(initialTab === undefined ? {} : { initialTab })}
      />
    </I18nProvider>
  )
  const { rerender } = render(view(true, 'launch'))
  expect(screen.getByRole('tab', { selected: true }).textContent).toBe('Launch')
  rerender(view(false, 'launch'))
  rerender(view(true))
  expect(screen.getByRole('tab', { selected: true }).textContent).toBe('Appearance')
})
