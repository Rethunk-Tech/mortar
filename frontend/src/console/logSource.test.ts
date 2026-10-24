import { expect, test } from 'bun:test'
import {
  type Entry,
  Level,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import { countByLevel } from './filter.ts'
import { mask, sourceOf, withoutHidden } from './logSource.ts'

const line = (message: string, extra: Partial<Entry> = {}): Entry => ({
  seq: 0,
  time: '',
  level: Level.Warn,
  mod: 'Unity Log',
  message,
  cont: false,
  ...extra,
})

const BOX = 'BoxCollider does not support negative scale or size.'
const box = (path: string) => [line(BOX), line(`Scene hierarchy path "${path}"`, { cont: true })]

test('numbers are masked so one call matches every time it logs', () => {
  expect(mask('Could not build 3')).toBe('Could not build #')
  expect(mask('[Dissonance:Recording] (16:34:55.653) Started\nsecond line')).toBe(
    '[Dissonance:Recording] (#) Started',
  )
  expect(mask('-1')).toBe('#')
})

test('a hidden line leaves the log and its counts, and nothing else from the same method', () => {
  const log = [
    line('1', { source: 'Round::Start', level: Level.Info }),
    line('Could not build 3', { source: 'Round::Start' }),
    line('Could not build 7', { source: 'Round::Start' }),
    line('1', { source: 'Terminal::Week', level: Level.Info }),
    ...box('Environment/Ship/Switch'),
    ...box('Environment/Ship/Door'),
    line('Loaded', { mod: 'MyPlugin', level: Level.Info }),
  ]
  const build = sourceOf(log.slice(1, 2))
  expect(build).toEqual({
    source: 'Round::Start',
    text: 'Could not build #',
    object: '',
    label: 'Could not build 3',
  })
  expect(sourceOf([line('Loaded', { mod: 'MyPlugin' })])).toBeNull()
  const door = sourceOf(log.slice(6, 8))
  const shown = withoutHidden(log, [
    { ...(build ?? { source: '', label: '' }) },
    { ...(door ?? { source: '', label: '' }) },
  ])
  expect(shown.map((e) => `${e.source ?? ''}:${e.message}`)).toEqual([
    'Round::Start:1',
    'Terminal::Week:1',
    `:${BOX}`,
    ':Scene hierarchy path "Environment/Ship/Switch"',
    ':Loaded',
  ])
  expect(countByLevel(shown).get(Level.Warn)).toBe(2)
  // Every message from the method: both builds and its bare "1".
  expect(
    withoutHidden(log, [{ source: 'Round::Start', text: '', object: '', label: '' }]).filter(
      (e) => e.source === 'Round::Start',
    ),
  ).toEqual([])
  // Every object: both colliders go.
  const every = withoutHidden(log, [{ ...(door ?? { source: '', label: '' }), object: '' }])
  expect(every.some((e) => e.message === BOX)).toBe(false)
})
