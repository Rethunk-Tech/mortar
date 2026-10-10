import { expect, test } from 'bun:test'
import { useProfiles } from '../profiles/store.ts'
import { enableRequirementsMode } from './enableRequirementsApply.ts'

test("the open profile's own setting decides what happens to requirements, over the game's", () => {
  const before = useProfiles.getState()
  try {
    const game = enableRequirementsMode()
    const own = game === 'never' ? 'ask' : 'never'
    useProfiles.setState({
      openId: 'p1',
      profiles: [
        {
          ...({} as (typeof before.profiles)[number]),
          id: 'p1',
          overrides: { enableRequirements: own },
        },
      ],
    })
    expect(enableRequirementsMode()).toBe(own)
    useProfiles.setState({ openId: 'other' })
    expect(enableRequirementsMode()).toBe(game)
  } finally {
    useProfiles.setState(before)
  }
})
