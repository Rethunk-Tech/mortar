import { beforeEach, expect, mock, test } from 'bun:test'
import type { Target } from './store.ts'

const asked: string[] = []
mock.module('./api.ts', () => ({
  configApi: {
    files: async () => [
      { name: '(in-game menu)', label: 'In-game menu', format: 'gmcm', sections: [] },
    ],
    schema: async (_t: Target, file: string) => {
      asked.push(file)
      return []
    },
  },
}))

const { useTypedConfig } = await import('./store.ts')

const target: Target = { game: 'stardew', profile: 'p', key: 'k', id: 'Seed.Beta' }

beforeEach(() => {
  asked.length = 0
  useTypedConfig.setState({ target: null, files: [], current: '', errors: {}, loadError: '' })
})

test('a mod with only an in-game menu opens it and never asks for config.json', async () => {
  const { open, select } = useTypedConfig.getState()
  await open(target)
  // The pane's effect can still hold the previous mod's file name when it fires.
  await select('config.json')
  expect(asked).toEqual(['(in-game menu)'])
  expect(useTypedConfig.getState().current).toBe('(in-game menu)')
  expect(useTypedConfig.getState().loadError).toBe('')
})
