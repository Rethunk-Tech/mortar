import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

test('launch and settings use the shared FlatpakGrant', () => {
  const launch = readFileSync(join(import.meta.dir, '..', 'launch', 'LaunchLayer.tsx'), 'utf8')
  const settings = readFileSync(
    join(import.meta.dir, '..', 'settings', 'sections', 'GameSettings.tsx'),
    'utf8',
  )
  expect(launch).toContain("from '../shell/FlatpakGrant.tsx'")
  expect(launch).not.toContain('function FlatpakGrant')
  expect(settings).toContain("from '../../shell/FlatpakGrant.tsx'")
  expect(settings).not.toContain('function FlatpakAccess')
})
