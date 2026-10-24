import { expect, test } from 'bun:test'
import type { MouseEvent } from 'react'
import { titleMenuTrigger, useOpenTitleMenu } from './titleMenus.ts'

const on = (el: HTMLElement) => ({ currentTarget: el }) as MouseEvent<HTMLElement>

test('a click on another title bar trigger while a menu is open switches menus at once', () => {
  const games = {} as HTMLElement
  const app = {} as HTMLElement
  titleMenuTrigger('game').onClick(on(games))
  titleMenuTrigger('app').onClick(on(app))
  expect(useOpenTitleMenu.getState()).toMatchObject({ id: 'app', anchor: app })
  titleMenuTrigger('app').onClick(on(app))
  expect(useOpenTitleMenu.getState().id).toBeNull()
})

test('moving over another trigger switches only while a menu is open, and the click after it keeps it open', () => {
  const el = {} as HTMLElement
  titleMenuTrigger('profile').onMouseEnter(on(el))
  expect(useOpenTitleMenu.getState().id).toBeNull()
  titleMenuTrigger('game').onClick(on(el))
  titleMenuTrigger('profile').onMouseEnter(on(el))
  titleMenuTrigger('profile').onClick(on(el))
  expect(useOpenTitleMenu.getState()).toMatchObject({ id: 'profile', byHover: false })
})
