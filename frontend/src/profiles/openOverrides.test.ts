import { expect, test } from 'bun:test'
import { openOverride } from './openOverrides.ts'
import { useProfiles } from './store.ts'

test('a setting reads the open profile override, else the game value', () => {
  const before = useProfiles.getState()
  try {
    useProfiles.setState({
      openId: 'p1',
      profiles: [
        {
          ...({} as (typeof before.profiles)[number]),
          id: 'p1',
          overrides: { consoleLevel: 'debug' },
        },
        {
          ...({} as (typeof before.profiles)[number]),
          id: 'p2',
          overrides: { consoleLevel: 'error' },
        },
      ],
    })
    expect(openOverride('consoleLevel', 'warn')).toBe('debug')
    expect(openOverride('cosmeticConflicts', 'collapsed')).toBe('collapsed')
    useProfiles.setState({ openId: 'p3' })
    expect(openOverride('consoleLevel', 'warn')).toBe('warn')
  } finally {
    useProfiles.setState(before)
  }
})
