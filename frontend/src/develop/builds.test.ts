import { afterEach, expect, mock, test } from 'bun:test'
import * as devsvc from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { useSettings } from '../settings/store.ts'
import { stubBinding } from '../testing/stubBinding.ts'
import { buildProfileProjects, useDevRuns, useProjectInfo } from './builds.ts'

afterEach(() => mock.restore())

function freshStart(inspect: { builtAt: number; stale: boolean }) {
  useDevRuns.setState({ byDir: {} })
  useProjectInfo.setState({ byDir: {} })
  useSettings.setState({
    games: { g: { devProjects: [{ dir: '/mod', name: 'Mod', profile: 'p' }] } },
  } as never)
  stubBinding(devsvc, 'Inspect', () => inspect)
  stubBinding(devsvc, 'BuildOrder', (dirs: never) => [dirs])
  return stubBinding(devsvc, 'Build', () => ({ ok: true }))
}

test('a send right after a start rebuilds only what its files say changed', async () => {
  const built = freshStart({ builtAt: 5, stale: false })
  expect(await buildProfileProjects('g', 'p')).toBeNull()
  expect(built).not.toHaveBeenCalled()

  const rebuilt = freshStart({ builtAt: 5, stale: true })
  expect(await buildProfileProjects('g', 'p')).toBeNull()
  expect(rebuilt).toHaveBeenCalledTimes(1)
})
