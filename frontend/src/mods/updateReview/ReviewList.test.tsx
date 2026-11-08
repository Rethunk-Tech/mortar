import { afterEach, expect, test } from 'bun:test'
import { I18nProvider } from '@lingui/react'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { useState } from 'react'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { Profile } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { i18n } from '../../i18n/index.ts'
import { acknowledgeUpdateCaution } from '../../launch/autoUpdate.ts'
import { testProfile } from '../../profiles/testProfile.ts'
import { ReviewList } from './ReviewList.tsx'

afterEach(() => {
  cleanup()
})

const profile = testProfile({ id: 'p', name: 'P' })
const update = (version: string): Update =>
  ({
    key: 'k',
    id: 'A',
    name: 'A',
    installed: '1',
    version,
    cautionMessage: 'Old saves may break.',
    url: '',
  }) as Update

// The dialog redraws the list after an acknowledgement, as it does in the review.
function List({ u, p }: { u: Update; p: Profile }) {
  const [n, setN] = useState(0)
  return (
    <I18nProvider i18n={i18n}>
      <ReviewList key={n} list={[u]} profile={p} onRefresh={() => setN(n + 1)} />
    </I18nProvider>
  )
}

const renderList = (u: Update) => render(<List u={u} p={profile} />)
const box = () => screen.getByRole('checkbox') as HTMLInputElement

test('an acknowledgement shows again after the list remounts', () => {
  i18n.activate('en')
  const u = update('2')
  const { unmount } = renderList(u)
  fireEvent.click(box())
  expect(box().checked).toBe(true)
  unmount()
  renderList(u)
  expect(box().checked).toBe(true)
  acknowledgeUpdateCaution(profile.id, u, false)
})

test('an acknowledgement of one version does not count for the next', () => {
  i18n.activate('en')
  acknowledgeUpdateCaution(profile.id, update('2'), true)
  renderList(update('3'))
  expect(box().checked).toBe(false)
  acknowledgeUpdateCaution(profile.id, update('2'), false)
})
