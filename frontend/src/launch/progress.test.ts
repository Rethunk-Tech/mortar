import { expect, test } from 'bun:test'
import {
  type Progress,
  State,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { nextPhase } from './progress.ts'

const loading: Progress = {
  game: 'g',
  profile: 'p',
  via: '',
  handed: false,
  copies: 1,
  up: 1,
  loader: 'bepinex5',
  tracks: true,
  readies: true,
  loaderStarted: true,
  loaded: 3,
  total: 40,
  mod: 'x',
  ready: false,
  usualMs: 0,
}

test('a running game keeps the screen up while its loader loads, then says it is running', () => {
  expect(nextPhase('launching', State.Running, loading)).toBe('loading')
  expect(nextPhase('launching', State.Running, { ...loading, ready: true })).toBe('running')
  expect(nextPhase('launching', State.Running, { ...loading, readies: false })).toBe('running')
  expect(nextPhase('launching', State.Running, null)).toBe('running')
})

test('a failure stays on the screen, and a cancelled or closed launch takes it down', () => {
  expect(nextPhase('loading', State.Failed, loading)).toBe('failed')
  expect(nextPhase('failed', State.Idle, loading)).toBe('failed')
  expect(nextPhase('launching', State.Idle, loading)).toBe('closed')
  expect(nextPhase('closed', State.Running, loading)).toBe('closed')
})
