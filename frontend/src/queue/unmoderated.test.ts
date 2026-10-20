import { expect, test } from 'bun:test'
import { useSettings } from '../settings/store.ts'
import { unwarnedSources } from './unmoderated.ts'

test('only unmoderated sites the player was not warned about yet are asked about, once each', () => {
  useSettings.setState({ tipsSeen: ['source-warning:itch'] })
  expect(unwarnedSources(['github', 'nexus', 'github', 'itch', 'thunderstore'])).toEqual(['github'])
})
