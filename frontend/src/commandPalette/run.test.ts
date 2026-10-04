import { beforeEach, expect, mock, test } from 'bun:test'

const load = mock(() => Promise.resolve())
const check = mock(() => Promise.resolve())

// The real stores get mock actions: mock.module would replace these modules for every later test file in the run.
const { useUpdates } = await import('../mods/updates.ts')
const { useMortarUpdate } = await import('../settings/updates.ts')
const { runPaletteItem } = await import('./run.ts')

beforeEach(() => {
  load.mockClear()
  check.mockClear()
  useUpdates.setState({ load })
  useMortarUpdate.setState({ check })
})

test('the Check for mod updates action loads profile mod updates', () => {
  runPaletteItem('action:updates')
  expect(load).toHaveBeenCalled()
  expect(check).not.toHaveBeenCalled()
})
