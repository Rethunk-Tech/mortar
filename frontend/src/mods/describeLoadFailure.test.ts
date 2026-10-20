import { expect, test } from 'bun:test'
import type { Result } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { describeLoadFailure } from './describe.ts'
import { problemCount } from './lookup.ts'
import { loadKindText } from './problemText.ts'

const row = (kind: string, message: string) => ({
  key: '',
  id: '',
  name: '',
  plugin: 'ModsEnabled',
  kind,
  message,
  line: 0,
})

test('a game-setting row is titled "Game setting off" and described by the catalog message', () => {
  expect(loadKindText('game-setting')).toBe('Game setting off')
  expect(describeLoadFailure(row('game-setting', 'Custom content is off in Options.ini.'))).toBe(
    'Custom content is off in Options.ini.',
  )
})

test('other load failures keep their sentences', () => {
  expect(describeLoadFailure(row('load-exception', ''))).toBe('ModsEnabled failed to load.')
  expect(describeLoadFailure(row('run-failed', 'no display'))).toBe(
    'The last launch failed: no display.',
  )
})

test('a plugin that loaded and then threw says so, and is not counted as a problem', () => {
  const ran = { ...row('unity-exception', 'NullReferenceException'), ran: true, errors: 2 }
  expect(describeLoadFailure(ran)).toBe('ModsEnabled loaded and logged 2 errors while running.')
  const result = { loadFailures: [ran, row('load-exception', '')] } as unknown as Result
  expect(problemCount(result)).toBe(1)
})

test('a ran error names where it was thrown and the other mods that patch it', () => {
  const ran = { ...row('unity-exception', 'NullReferenceException'), ran: true, errors: 1 }
  expect(
    describeLoadFailure({ ...ran, thrown: 'OccludeAudio.Update', patchedBy: ['A', 'B'] }),
  ).toBe(
    'ModsEnabled loaded and logged an error while running. It was thrown in OccludeAudio.Update, which A and B also patch.',
  )
  expect(describeLoadFailure({ ...ran, thrown: 'OccludeAudio.Update' })).toBe(
    'ModsEnabled loaded and logged an error while running. It was thrown in OccludeAudio.Update.',
  )
})

test("a run's errors count only when the run crashed", () => {
  const runError = { key: 'k', id: 'x', name: 'X', count: 1, first: '', runId: 'r', updated: false }
  const calm = { runErrors: [{ ...runError, severe: false }] } as unknown as Result
  const crashed = { runErrors: [{ ...runError, severe: true }] } as unknown as Result
  expect([problemCount(calm), problemCount(crashed)]).toEqual([0, 1])
})
