import { expect, test } from 'bun:test'
import { closeTitleMenu, useOpenTitleMenu } from './titleMenus.ts'

const el = (name: string) => ({ name }) as unknown as HTMLElement

test('opening a title menu replaces the open one, as in a menu bar', () => {
  closeTitleMenu()
  useOpenTitleMenu.setState({ id: 'game', anchor: el('game') })
  useOpenTitleMenu.setState({ id: 'profile', anchor: el('profile') })
  expect(useOpenTitleMenu.getState().id).toBe('profile')
  closeTitleMenu()
  expect(useOpenTitleMenu.getState()).toEqual({ id: null, anchor: null })
})
