import { expect, mock, test } from 'bun:test'

mock.module('@lingui/core/macro', () => ({
  msg: (parts: TemplateStringsArray) => parts.join(''),
  plural: () => '',
}))

const { useSettings } = await import('../settings/store.ts')
const { useToasts } = await import('../toasts/store.ts')
const { showDigest } = await import('./updateDigestToast.ts')

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
