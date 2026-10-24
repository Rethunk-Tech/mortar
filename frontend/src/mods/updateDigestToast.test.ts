import { afterEach, expect, test } from 'bun:test'
import { useSettings } from '../settings/store.ts'
import { useToasts } from '../toasts/store.ts'
import { showDigest } from './updateDigestToast.ts'

afterEach(() => {
  useToasts.setState(useToasts.getInitialState(), true)
})

const notice = {
  game: 'stardew',
  profile: 'p',
  openProfiles: false,
  totalUpdates: 2,
  profilesWith: 1,
}

test('the update digest toasts only while In Mortar notices for mod updates are on', () => {
  useToasts.setState({ toasts: [] })
  useSettings.setState({ notifyModUpdates: false })
  showDigest(notice)
  expect(useToasts.getState().toasts).toHaveLength(0)
  useSettings.setState({ notifyModUpdates: true })
  showDigest(notice)
  expect(useToasts.getState().toasts).toHaveLength(1)
})
