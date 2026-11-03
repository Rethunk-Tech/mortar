import { expect, test } from 'bun:test'
import { useProfiles } from '../profiles/store.ts'
import { useToasts } from '../toasts/store.ts'
import { announceUpdates, useUpdates } from './updates.ts'

test('the update review stays open across a tab switch and closes when another profile opens', () => {
  useProfiles.setState({ openId: 'a' })
  useUpdates.getState().setReviewing(true)
  useProfiles.setState({ openId: 'a', profiles: [] })
  expect(useUpdates.getState().reviewing).toBe(true)
  useProfiles.setState({ openId: 'b' })
  expect(useUpdates.getState().reviewing).toBe(false)
})

test('a newer updates toast replaces the one still showing', () => {
  useToasts.setState(useToasts.getInitialState(), true)
  announceUpdates(2)
  announceUpdates(3)
  const shown = useToasts.getState().toasts
  expect(shown).toHaveLength(1)
  expect(shown[0]?.title).toContain('3')
})
