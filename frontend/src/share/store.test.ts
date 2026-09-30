import { beforeEach, expect, test } from 'bun:test'
import { useNav } from '../nav/store.ts'
import { useNexus } from '../settings/nexus.ts'
import { importAfterSignIn, useImportDialog } from './store.ts'

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
