import { expect, test } from 'bun:test'
import { profileById, profileByIdIncludingDamaged } from './store.ts'
import { testProfile } from './testProfile.ts'

test('profileByIdIncludingDamaged finds a damaged profile that profileById skips', () => {
  const live = testProfile({ id: 'live', name: 'Live' })
  const broken = testProfile({ id: 'dead', name: 'Dead' })
  const s = { profiles: [live], damaged: [broken] }
  expect(profileById('dead')(s)).toBeUndefined()
  expect(profileByIdIncludingDamaged('dead')(s)?.name).toBe('Dead')
  expect(profileByIdIncludingDamaged('live')(s)?.name).toBe('Live')
})
