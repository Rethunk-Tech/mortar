import { expect, test } from 'bun:test'
import { Outcome } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import type { Run } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { bisectBlock, bisectFailure } from './bisectBlock.ts'

const run = (over: Record<string, unknown> = {}) =>
  ({ outcome: Outcome.OutcomeRan, errors: 0, ...over }) as Run
const crashed = { outcome: Outcome.OutcomeCrashed }

test('a crash check needs the newest run to have crashed with no known cause', () => {
  expect(bisectBlock([])).toBe('no-runs')
  expect(bisectBlock(null)).toBe('no-runs')
  expect(bisectBlock([run(), run(crashed)])).toBe('ended-normally')
  expect(bisectBlock([run(crashed)])).toBeNull()
  expect(bisectBlock([run({ errors: 2, exit: { stopped: false } })])).toBeNull()
  expect(bisectBlock([run({ errors: 2, exit: { stopped: true } })])).toBe('ended-normally')
  expect(bisectBlock([run({ ...crashed, cause: { modName: 'X' } })])).toBe('cause-known')
})

test('crash check failures are told apart by their Go text', () => {
  expect(bisectFailure('the profile has no enabled user mods')).toBe('no-mods')
  expect(bisectFailure('mod folder [FS]Kyuya hats Pack is missing')).toBe('missing-files')
  expect(bisectFailure('launch failed')).toBe('other')
  expect(bisectFailure(undefined)).toBe('other')
})
