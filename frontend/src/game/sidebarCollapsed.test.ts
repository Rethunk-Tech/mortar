import { expect, test } from 'bun:test'
import { readCollapsed, useSidebarCollapsed } from './sidebarCollapsed.ts'

test('the collapsed choice is saved on toggle and read back, expanded when nothing was saved', () => {
  const saved = new Map<string, string>()
  globalThis.localStorage = {
    getItem: (k: string) => saved.get(k) ?? null,
    setItem: (k: string, v: string) => saved.set(k, v),
  } as unknown as Storage
  expect(readCollapsed()).toBe(false)
  useSidebarCollapsed.setState({ collapsed: false })
  useSidebarCollapsed.getState().toggle()
  expect(useSidebarCollapsed.getState().collapsed).toBe(true)
  expect(readCollapsed()).toBe(true)
  useSidebarCollapsed.getState().toggle()
  expect(readCollapsed()).toBe(false)
  saved.set('mortar.sidebarCollapsed', '"yes"')
  expect(readCollapsed()).toBe(false)
})
