import { beforeEach, expect, mock, test } from 'bun:test'

const SHARE_SERVICE = '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/service.ts'
const realShare = await import(SHARE_SERVICE)
mock.module(SHARE_SERVICE, () => ({ ...realShare, Discard: () => Promise.resolve() }))

const { usePatreonPost } = await import('../install/patreon.ts')
const { useNav } = await import('../nav/store.ts')
const { useNexus } = await import('../settings/nexus.ts')
const { importAfterSignIn, useImportDialog } = await import('./store.ts')

beforeEach(() => {
  useNav.setState(useNav.getInitialState(), true)
  useNexus.setState(useNexus.getInitialState(), true)
  useImportDialog.setState(useImportDialog.getInitialState(), true)
})

const signOut = () => useNexus.setState({ signedIn: false })
const signIn = () => useNexus.setState({ signedIn: true })

test('import reopens with the same link once the sign-in succeeds', () => {
  signOut()
  useNav.getState().openGame('stardew')
  importAfterSignIn({ profileId: 'p1', link: 'mortar://stardew/p/abc' })
  expect(useNav.getState().route.name).toBe('settings')
  expect(useImportDialog.getState().request).toBeNull()
  signIn()
  expect(useImportDialog.getState().request).toMatchObject({
    profileId: 'p1',
    tab: 'link',
    seed: 'mortar://stardew/p/abc',
  })
})

test('leaving settings without signing in cancels the return to import', () => {
  signOut()
  useNav.getState().openGame('stardew')
  importAfterSignIn({ link: 'x' })
  useNav.getState().closeSettings()
  signIn()
  expect(useImportDialog.getState().request).toBeNull()
})

test('closing the dialog forgets a remembered Patreon post', () => {
  usePatreonPost.getState().set({ id: '4242', url: 'https://www.patreon.com/posts/4242' })
  useImportDialog.getState().close()
  expect(usePatreonPost.getState().post).toBeNull()
})
