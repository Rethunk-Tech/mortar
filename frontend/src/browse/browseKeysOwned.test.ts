import { afterEach, expect, test } from 'bun:test'
import { browseKeysOwned } from './selection.ts'

afterEach(() => {
  document.body.replaceChildren()
})

function el(html: string): HTMLElement {
  document.body.innerHTML = html
  return document.body.firstElementChild as HTMLElement
}

test('text fields own Browse keys', () => {
  expect(browseKeysOwned(el('<input type="text">'))).toBe(true)
  expect(browseKeysOwned(el('<textarea></textarea>'))).toBe(true)
  expect(browseKeysOwned(el('<select></select>'))).toBe(true)
})

test('checkbox, radio and button inputs do not own Browse keys', () => {
  expect(browseKeysOwned(el('<input type="checkbox">'))).toBe(false)
  expect(browseKeysOwned(el('<input type="radio">'))).toBe(false)
  expect(browseKeysOwned(el('<input type="button">'))).toBe(false)
})

test('an open dialog owns Browse keys', () => {
  document.body.innerHTML = '<div role="dialog"></div><button type="button"></button>'
  expect(browseKeysOwned(document.querySelector('button'))).toBe(true)
})

test('a menu owns Browse keys', () => {
  document.body.innerHTML = '<div role="menu"><button type="button"></button></div>'
  expect(browseKeysOwned(document.querySelector('button'))).toBe(true)
})
