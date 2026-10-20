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
