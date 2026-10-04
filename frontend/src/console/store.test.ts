import { expect, test } from 'bun:test'
import { Hint } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import {
  State,
  type Status,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { canSendTo } from './store.ts'

const status = (profile: string, state = State.Running): Status => ({
  game: 'stardew',
  state,
  profile,
  since: 0,
  hint: Hint.$zero,
  error: '',
})

test('commands go only to the open profile that is running', () => {
  expect(canSendTo(status('a'), 'stardew', 'a')).toBe(true)
  expect(canSendTo(status('b'), 'stardew', 'a')).toBe(false)
  expect(canSendTo(status('a', State.Launching), 'stardew', 'a')).toBe(false)
  expect(canSendTo(null, 'stardew', 'a')).toBe(false)
})
