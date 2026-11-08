import { afterEach, expect, test } from 'bun:test'
import { I18nProvider } from '@lingui/react'
import { cleanup, render, screen } from '@testing-library/react'
import type { UpdatesResult } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { i18n } from '../../i18n/index.ts'
import { useProfiles } from '../../profiles/store.ts'
import { testProfile } from '../../profiles/testProfile.ts'
import { useBadges } from '../badges.ts'
import { useNexusDetails } from '../nexusDetails.ts'
import { useUpdates } from '../updates.ts'
import { UpdateBar } from './UpdateBar.tsx'

afterEach(() => {
  cleanup()
  useUpdates.setState({ updates: null })
})

test('the teaser names an update the review offers, not one it withholds', () => {
  i18n.activate('en')
  const profile = testProfile({ id: 'p', name: 'P' })
  // The profile first: opening another profile clears the updates the store holds.
  useProfiles.setState({ game: { id: 'stardew' }, openId: 'p', profiles: [profile] } as never)
  // A sits on a hidden Nexus page, so the count and the review offer only B, though A comes first.
  useUpdates.setState({
    updates: {
      updates: [
        { key: 'a', id: 'A', name: 'Hidden', version: '2', nexusId: 7 },
        { key: 'b', id: 'B', name: 'Offered', version: '3', nexusId: 0 },
      ],
    } as UpdatesResult,
  })
  useBadges.setState({ byProfile: { p: { updates: 1, problems: 0, missing: 0 } } })
  useNexusDetails.setState({
    byKey: { 'stardew:7': { details: { page: { status: 'published', available: false } } } },
  } as never)
  render(
    <I18nProvider i18n={i18n}>
      <UpdateBar />
    </I18nProvider>,
  )
  expect(screen.getByText(/including/).textContent).toContain('Offered')
})
