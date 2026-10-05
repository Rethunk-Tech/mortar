import { expect, test } from 'bun:test'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { lastRunOf, useLastRun } from './lastRun.ts'

function mod(id: string): Mod {
  return {
    key: 'k',
    id,
    name: 'Content Patcher',
    author: '',
    version: '',
    enabled: true,
    siblings: [],
    picture: '',
    endorsements: 0,
  }
}

test('last-run badges clear when the store is reset to the initial state', () => {
  useLastRun.setState(useLastRun.getInitialState(), true)
  const id = 'Pathoschild.ContentPatcher'
  useLastRun.setState({
    runId: 'r1',
    byId: {
      [id.toLowerCase()]: { name: 'Content Patcher', id, errors: 2, warnings: 1 },
    },
  })
  expect(lastRunOf(mod(id))?.errors).toBe(2)
  expect(lastRunOf(mod(id.toLowerCase()))?.errors).toBe(2)
  expect(lastRunOf(mod('pathoschild.contentpatcher'))?.errors).toBe(2)
  useLastRun.setState(useLastRun.getInitialState(), true)
  expect(lastRunOf(mod(id))).toBeUndefined()
})
