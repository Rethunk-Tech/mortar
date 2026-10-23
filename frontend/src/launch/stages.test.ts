import { expect, test } from 'bun:test'
import type { Progress } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { launchStages, waitsForLoader } from './stages.ts'

const base: Progress = {
  game: 'lethal-company',
  profile: 'p',
  via: 'steam',
  handed: false,
  copies: 1,
  up: 0,
  loader: 'bepinex5',
  tracks: true,
  readies: true,
  loaderStarted: false,
  loaded: 0,
  total: 0,
  mod: '',
  ready: false,
  usualMs: 0,
}

const states = (p: Partial<Progress>, failed = false) =>
  launchStages({ ...base, ...p }, failed).map((s) => `${s.id}:${s.state}`)

test('each mark is done once the launch has passed it, and the next is the one in progress', () => {
  expect(states({})).toEqual(['via:active', 'game:pending', 'loader:pending'])
  expect(states({ handed: true })).toEqual(['via:done', 'game:active', 'loader:pending'])
  expect(states({ handed: true, up: 1, loaderStarted: true })).toEqual([
    'via:done',
    'game:done',
    'loader:active',
  ])
  expect(states({ handed: true, up: 1, loaderStarted: true, ready: true })).toEqual([
    'via:done',
    'game:done',
    'loader:done',
  ])
})

test('a direct start has no launcher mark, a launch without mods no loader mark, and copies all come up first', () => {
  expect(states({ via: '', loader: '', copies: 3, up: 2 })).toEqual(['game:active'])
  expect(states({ via: '', loader: '', copies: 3, up: 3 })).toEqual(['game:done'])
})

test('a failed launch fails the mark it was on', () => {
  expect(states({ handed: true }, true)).toEqual(['via:done', 'game:failed', 'loader:pending'])
})

test('the screen waits for a loader that says when it is done, and only for one', () => {
  expect(waitsForLoader({ ...base, loaderStarted: true })).toBe(true)
  expect(waitsForLoader({ ...base, readies: false, loaderStarted: true })).toBe(false)
  expect(states({ handed: true, up: 1, loaderStarted: true, readies: false })).toEqual([
    'via:done',
    'game:done',
    'loader:done',
  ])
  expect(waitsForLoader({ ...base, loader: '' })).toBe(false)
})
