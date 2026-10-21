import { expect, test } from 'bun:test'
import { isInstalledVersion, targetOfSource, versionChoices } from './versionChoices.ts'

test('a held mod gets an exact update that replaces its entry, whatever the source', () => {
  const [nexus] = versionChoices(
    { source: 'nexus', id: '12' },
    [{ version: '1.0', fileId: 7, note: 'Main' }],
    'k',
  )
  expect(nexus?.want).toEqual({
    kind: 'update',
    currentKey: 'k',
    exact: true,
    modId: 12,
    fileId: 7,
    version: '1.0',
  })
  const [github] = versionChoices({ source: 'github', id: 'a/b' }, [{ version: '2.0' }], '')
  expect(github?.want).toEqual({ kind: 'install', repo: 'a/b', tag: '2.0', version: '2.0' })
  const [direct] = versionChoices({ source: 'modrinth', id: 'p' }, [{ version: '3' }], '')
  expect(direct?.want).toMatchObject({ source: 'modrinth', package: 'p', version: '3' })
})

test('the installed version is told apart by its number, a mod without a site has no target', () => {
  const [c] = versionChoices({ source: 'thunderstore', id: 'A-B' }, [{ version: 'v1.2.0' }], '')
  expect(c && isInstalledVersion(c, '1.2')).toBe(true)
  expect(targetOfSource({ kind: 'local', name: 'x' })).toBeNull()
  expect(targetOfSource({ kind: 'nexus', name: 'x', modId: 5 })).toEqual({
    source: 'nexus',
    id: '5',
  })
})
