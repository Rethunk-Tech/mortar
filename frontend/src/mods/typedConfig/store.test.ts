import { afterEach, beforeEach, expect, mock, spyOn, test } from 'bun:test'
import { configApi } from './api.ts'
import { type Target, useTypedConfig } from './store.ts'

const asked: string[] = []

const target: Target = { game: 'stardew', profile: 'p', key: 'k', id: 'Seed.Beta' }

beforeEach(() => {
  asked.length = 0
  spyOn(configApi, 'files').mockResolvedValue([
    { name: '(in-game menu)', label: 'In-game menu', format: 'gmcm', sections: [] },
  ])
  spyOn(configApi, 'schema').mockImplementation(async (_t, file) => {
    asked.push(file)
    return []
  })
  useTypedConfig.setState({ target: null, files: [], current: '', errors: {}, loadError: '' })
})
afterEach(() => {
  mock.restore()
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
