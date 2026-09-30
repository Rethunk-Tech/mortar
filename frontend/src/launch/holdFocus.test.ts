import { expect, test } from 'bun:test'
import { holdFocus, launchEscHides } from './holdFocus.ts'

interface Fake {
  inert: boolean
  kids: Fake[]
  parentElement: Fake | null
  id: string
  dataset: { windowControls?: string }
  readonly children: Fake[]
  querySelector: (sel: string) => Fake | null
  contains: (other: Fake) => boolean
  append: (...kids: Fake[]) => void
}

function el(): Fake {
  const node: Fake = {
    inert: false,
    kids: [],
    parentElement: null,
    id: '',
    dataset: {},
    get children() {
      return node.kids
    },
    querySelector(sel: string) {
      const walk = (n: Fake): Fake | null => {
        if (sel === '[data-window-controls]' && n.dataset.windowControls !== undefined) {
          return n
        }
        for (const c of n.kids) {
          const hit = walk(c)
          if (hit) {
            return hit
          }
        }
        return null
      }
      return walk(node)
    },
    contains(other: Fake) {
      if (other === node) {
        return true
      }
      return node.kids.some((c) => c === other || c.contains(other))
    },
    append(...kids: Fake[]) {
      for (const k of kids) {
        k.parentElement = node
        node.kids.push(k)
      }
    },
  }
  return node
}

const asEl = (n: Fake) => n as unknown as HTMLElement

test('holdFocus inerts siblings and title-bar chrome but not window controls', () => {
  const root = el()
  root.id = 'root'
  const chrome = el()
  const tab = el()
  const menu = el()
  const controls = el()
  controls.dataset.windowControls = ''
  const close = el()
  controls.append(close)
  chrome.append(tab, menu, controls)
  const main = el()
  const content = el()
  const overlay = el()
  main.append(content, overlay)
  root.append(chrome, main)

  const restore = holdFocus(asEl(overlay))
  expect(content.inert).toBe(true)
  expect(tab.inert).toBe(true)
  expect(menu.inert).toBe(true)
  expect(controls.inert).toBe(false)
  expect(close.inert).toBe(false)
  restore()
  expect(content.inert).toBe(false)
  expect(tab.inert).toBe(false)
})

test('Esc hides the launch overlay unless another dialog is open', () => {
  expect(launchEscHides('Escape', 1)).toBe(true)
  expect(launchEscHides('Escape', 2)).toBe(false)
  expect(launchEscHides('Enter', 1)).toBe(false)
})
