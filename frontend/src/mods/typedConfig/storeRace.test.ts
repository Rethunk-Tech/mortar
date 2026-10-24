import { afterEach, expect, mock, spyOn, test } from 'bun:test'
import { configApi } from './api.ts'
import { type Target, useTypedConfig } from './store.ts'

afterEach(() => {
  mock.restore()
})

test('a mod opened while another still loads keeps the pane when the first answer arrives late', async () => {
  let releaseFirst: () => void = () => undefined
  spyOn(configApi, 'files').mockImplementation(async (t: Target) => {
    if (t.key === 'slow') {
      await new Promise<void>((resolve) => {
        releaseFirst = resolve
      })
      return [{ name: 'Slow.cfg', label: '', format: 'cfg', sections: [] }]
    }
    return [{ name: 'BepInEx.cfg', label: '', format: 'cfg', sections: [] }]
  })
  spyOn(configApi, 'schema').mockResolvedValue([])
  const { open } = useTypedConfig.getState()
  const first = open({ game: 'lethal-company', profile: 'p', key: 'slow', id: '' })
  await open({ game: 'lethal-company', profile: 'p', key: '', id: '' }, 'BepInEx.cfg')
  releaseFirst()
  await first
  const s = useTypedConfig.getState()
  expect(s.current).toBe('BepInEx.cfg')
  expect(s.files.map((f) => f.name)).toEqual(['BepInEx.cfg'])
})
