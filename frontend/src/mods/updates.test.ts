import { expect, test } from 'bun:test'
import { useProfiles } from '../profiles/store.ts'
import { useUpdates } from './updates.ts'

test('the update review stays open across a tab switch and closes when another profile opens', () => {
  useProfiles.setState({ openId: 'a' })
  useUpdates.getState().setReviewing(true)
  useProfiles.setState({ openId: 'a', profiles: [] })
  expect(useUpdates.getState().reviewing).toBe(true)
  useProfiles.setState({ openId: 'b' })
  expect(useUpdates.getState().reviewing).toBe(false)
})
