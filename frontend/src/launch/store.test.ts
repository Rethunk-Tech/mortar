import { beforeEach, expect, test } from 'bun:test'
import { Hint, Level } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'
import {
  type Lines,
  State,
  type Status,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { useConsole } from '../console/store.ts'
import { overlayGame, useLaunch } from './store.ts'

const status = (state: State, profile = 'p1'): Status => ({
  game: 'stardew',
  state,
  profile,
  since: 0,
  hint: Hint.$zero,
  error: '',
})

const line = (seq: number): Lines => ({
  game: 'stardew',
  profile: 'p1',
  entries: [{ seq, time: '', level: Level.Info, mod: '', message: '', cont: false }],
})

beforeEach(() => {
  useLaunch.setState(useLaunch.getInitialState(), true)
  useConsole.setState(useConsole.getInitialState(), true)
})

test('the launch overlay keeps the launching game when the route is not the game', () => {
  expect(overlayGame('game', 'stardew', '')).toBe('stardew')
  expect(overlayGame('settings', '', 'stardew')).toBe('stardew')
  expect(overlayGame('settings', '', '')).toBe('')
})

test('a launch shows its own log even when the console tab never loaded', () => {
  useLaunch.getState().apply(status(State.Launching))
  useConsole.getState().add(line(1))
  expect(useConsole.getState().entries).toHaveLength(1)
})

test('a refresh during the same launch keeps the log and a hidden overlay', () => {
  useLaunch.getState().apply(status(State.Launching))
  useConsole.getState().add(line(1))
  useLaunch.getState().hide()
  useLaunch.getState().apply(status(State.Launching))
  expect(useConsole.getState().entries).toHaveLength(1)
  expect(useLaunch.getState().hidden).toBe(true)
})

test('launching another profile starts over', () => {
  useLaunch.getState().apply(status(State.Launching))
  useConsole.getState().add(line(1))
  useLaunch.getState().hide()
  useLaunch.getState().apply(status(State.Launching, 'p2'))
  expect(useConsole.getState().entries).toHaveLength(0)
  expect(useLaunch.getState().hidden).toBe(false)
  expect(useConsole.getState().shown).toEqual({ game: 'stardew', profile: 'p2' })
})

test('a polled Idle keeps preparation going; an announced one ends it', () => {
  useLaunch.setState({ starting: true })
  useLaunch.getState().apply(status(State.Idle), true)
  expect(useLaunch.getState().starting).toBe(true)
  useLaunch.getState().apply(status(State.Idle))
  expect(useLaunch.getState().starting).toBe(false)
})
