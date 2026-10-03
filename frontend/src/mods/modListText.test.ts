import { expect, test } from 'bun:test'
import { formatModList } from './modListText.ts'

const enabled = {
  enabled: true,
  name: 'Lookup Anything',
  version: '1.50.0',
  nexusUrl: 'https://www.nexusmods.com/stardewvalley/mods/541',
}
const off = { enabled: false, name: 'Off', version: '2.0' }
const local = { enabled: true, name: 'Local pack', version: '0.1' }

test('markdown lists enabled mods with Nexus links and skips disabled ones', () => {
  expect(formatModList([enabled, off, local], 'markdown')).toBe(
    [
      '- [Lookup Anything 1.50.0](https://www.nexusmods.com/stardewvalley/mods/541)',
      '- Local pack 0.1',
    ].join('\n'),
  )
})

test('plain text lists enabled mods as name, version, and Nexus URL', () => {
  expect(formatModList([enabled, off, local], 'plain')).toBe(
    [
      'Lookup Anything 1.50.0 https://www.nexusmods.com/stardewvalley/mods/541',
      'Local pack 0.1',
    ].join('\n'),
  )
})
