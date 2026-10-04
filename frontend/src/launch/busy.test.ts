import { expect, test } from 'bun:test'
import {
  State,
  type Status,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { gameBusy } from './busy.ts'

const status = (state: State, game: string) => ({ state, game }) as Status

test('gameBusy is true only while launching or running, for the matching game', () => {
  expect(gameBusy(null)).toBe(false)
  expect(gameBusy(status(State.Idle, 'a'))).toBe(false)
  expect(gameBusy(status(State.Launching, 'a'))).toBe(true)
  expect(gameBusy(status(State.Running, 'a'), 'a')).toBe(true)
  expect(gameBusy(status(State.Running, 'a'), 'b')).toBe(false)
})
