import { beforeEach, expect, mock, test } from 'bun:test'

const load = mock(() => Promise.resolve())
const check = mock(() => Promise.resolve())

mock.module('../mods/updates.ts', () => ({
  useUpdates: { getState: () => ({ load }) },
}))

mock.module('../settings/updates.ts', () => ({
  useMortarUpdate: { getState: () => ({ check }) },
}))

const { runPaletteItem } = await import('./run.ts')

beforeEach(() => {
  load.mockClear()
  check.mockClear()
})

test('the Check for mod updates action loads profile mod updates', () => {
  runPaletteItem('action:updates')
  expect(load).toHaveBeenCalled()
  expect(check).not.toHaveBeenCalled()
})
