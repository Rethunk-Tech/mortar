import { expect, test } from 'bun:test'
import {
  type Entry,
  Level,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import { SHOW_ALL, visible } from './filter.ts'
import { sourceOf } from './logSource.ts'

const line = (message: string, extra: Partial<Entry> = {}): Entry => ({
  seq: 0,
  time: '',
  level: Level.Warn,
  mod: 'Unity Log',
  message,
  cont: false,
  ...extra,
})

test('two bare "1" lines from different call sites hide apart, and an engine warning hides per object', () => {
  const box = (path: string) => [
    line('BoxCollider does not support negative scale or size.'),
    line(`Scene hierarchy path "${path}"`, { cont: true }),
  ]
  const log = [
    line('1', { source: 'Round::Start@1a' }),
    line('1', { source: 'Terminal::Week@2c', object: 'Systems/Terminal' }),
    ...box('Environment/Ship/Switch'),
    ...box('Environment/Ship/Door'),
    line('Loaded', { mod: 'MyPlugin' }),
  ]
  expect(sourceOf(log.slice(2, 4))).toEqual({
    source: 'unity:BoxCollider does not support negative scale or size.',
    object: 'Environment/Ship/Switch',
    label: 'BoxCollider does not support negative scale or size.',
  })
  expect(sourceOf([line('Loaded', { mod: 'MyPlugin' })])).toBeNull()
  const shown = visible(log, SHOW_ALL, [
    { source: 'Round::Start@1a', object: '', label: '1' },
    {
      source: 'unity:BoxCollider does not support negative scale or size.',
      object: 'Environment/Ship/Door',
      label: '',
    },
  ])
  expect(shown.map((e) => e.source ?? e.message)).toEqual([
    'Terminal::Week@2c',
    'BoxCollider does not support negative scale or size.',
    'Scene hierarchy path "Environment/Ship/Switch"',
    'Loaded',
  ])
})
