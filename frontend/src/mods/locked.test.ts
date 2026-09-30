import { expect, test } from 'bun:test'
import { Hint } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'
import type { Status } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { State } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { isLocked } from './locked.ts'

const status = (state: State, profile: string): Status => ({
  game: 'stardew',
  state,
  profile,
  since: 0,
  hint: Hint.$zero,
  error: '',
})

test('only a launching or running game on the open profile locks it', () => {
  expect(isLocked(status(State.Launching, 'a'), 'a')).toBe(true)
  expect(isLocked(status(State.Running, 'a'), 'a')).toBe(true)
  expect(isLocked(status(State.Launching, 'b'), 'a')).toBe(false)
  expect(isLocked(status(State.Running, 'b'), 'a')).toBe(false)
  expect(isLocked(status(State.Idle, 'a'), 'a')).toBe(false)
  expect(isLocked(null, 'a')).toBe(false)
})
