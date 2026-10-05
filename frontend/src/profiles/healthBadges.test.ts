import { expect, test } from 'bun:test'
import { useHealthBadges } from './healthBadges.ts'

test('a check with findings shows the badge and a clean check clears it', () => {
  useHealthBadges.setState({ game: 'stardew', byProfile: {} })
  const { apply } = useHealthBadges.getState()
  apply({ game: 'stardew', profile: 'a', findings: 2 })
  apply({ game: 'other', profile: 'b', findings: 1 })
  expect(useHealthBadges.getState().byProfile).toEqual({ a: 2 })
  apply({ game: 'stardew', profile: 'a', findings: 0 })
  expect(useHealthBadges.getState().byProfile).toEqual({})
})
