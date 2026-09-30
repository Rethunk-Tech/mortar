import { expect, test } from 'bun:test'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { lastRunOf, useLastRun } from './lastRun.ts'

function mod(uniqueId: string): Mod {
  return {
    key: 'k',
    uniqueId,
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
  const uniqueId = 'Pathoschild.ContentPatcher'
  useLastRun.setState({
    runId: 'r1',
    byId: {
      [uniqueId]: { name: 'Content Patcher', uniqueId, errors: 2, warnings: 1 },
    },
  })
  expect(lastRunOf(mod(uniqueId))?.errors).toBe(2)
  useLastRun.setState(useLastRun.getInitialState(), true)
  expect(lastRunOf(mod(uniqueId))).toBeUndefined()
})
