import { expect, test } from 'bun:test'
import { useRenameRequest } from './renameRequest.ts'
import { useTab } from './tab.ts'

test('a rename request opens Home, where the name field is', () => {
  useTab.getState().setTab('mods')
  useRenameRequest.getState().request('p1')
  expect(useTab.getState().tab).toBe('home')
  expect(useRenameRequest.getState().id).toBe('p1')
})
